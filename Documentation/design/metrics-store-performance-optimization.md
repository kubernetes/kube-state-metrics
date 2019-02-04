# Kube-State-Metrics - Performance Optimization Proposal


---

Author: Max Inden (IndenML@gmail.com)

Date: 23. July 2018

Target release: v1.5.0

---


## Glossary

- kube-state-metrics: “Simple service that listens to the Kubernetes API server
  and generates metrics about the state of the objects”

- Time series: A single line in a /metrics response e.g.
  “metric_name{label="value"} 1”


## Problem Statement

There has been repeated reports of two issues running kube-state-metrics on
production Kubernetes clusters. First kube-state-metrics takes a long time
(“10s - 20s”) to respond on its /metrics endpoint, leading to Prometheus
instances dropping the scrape interval request and marking the given time series
as stale. Second kube-state-metrics uses a lot of memory and thereby being
out-of-memory killed due to low set Kubernetes resource limits.


## Goal

The goal of this proposal can be split into the following sub-goals ordered by
their priority:

1. Decrease response time on /metrics endpoint

2. Decrease overall runtime memory usage


## Status Quo

Instead of requesting the needed information from the Kubernetes API-Server on
demand (on scrape), kube-state-metrics uses the Kubernetes client-go cache tool
to keep a full in memory representation of all Kubernetes objects of a given
cluster. Using the cache speeds up the performance critical path of replying to
a scrape request, and reduces the load on the Kubernetes API-Server by only
sending deltas whenever they occur. Kube-state-metrics does not make use of all
properties and sub-objects of these Kubernetes objects that it stores in its
cache.

On a scrape request by e.g. Prometheus on the /metrics endpoint
kube-state-metrics calculates the configured time series on demand based on the
objects in its cache and converts them to the Prometheus string representation.


## Proposal

Instead of a full representation of all Kubernetes objects with all its
properties in memory via the Kubernetes client-go cache, use a map, addressable
by the Kubernetes object uuid, containing all time series of that object as a
single multi-line string.

```
var cache = map[uuid][]byte{}
```

Kube-state-metrics listens on add, update and delete events via Kubernetes
client-go reflectors. On add and update events kube-state-metrics generates all
time series related to the Kubernetes object based on the event’s payload,
concatenates the time series to a single byte slice and sets / replaces the byte
slice in the store at the uuid of the Kubernetes object. One can precompute the
length of a time series byte slice before allocation as the sum of the length of
the metric name, label keys and values as well as the metric value in string
representation. On delete events kube-state-metrics deletes the uuid entry of
the given Kubernetes object in the cache map.

On a scrape request on the /metrics endpoint, kube-state-metrics iterates over
the cache map and concatenates all time series string blobs into a single
string, which is finally passed on as a response.

```
       +---------------+ +-----------+                 +---------------+         +-------------------+
       | pod_reflector | | pod_store |                 | pod_collector |         | metrics_endpoint  |
       +---------------+ +-----------+                 +---------------+         +-------------------+
-------------\ |               |                               |                           |
| new pod p1 |-|               |                               |                           |
|------------| |               |                               |                           |
               |               |                               |                           |
               | Add(p1)       |                               |                           |
               |-------------->|                               |                           |
               |               | ----------------------\       |                           |
               |               |-| generateMetrics(p1) |       |                           |
               |               | |---------------------|       |                           |
               |               |                               |                           |
               |           nil |                               |                           |
               |<--------------|                               |                           |
               |               |                               |                           | ---------------\
               |               |                               |                           |-| GET /metrics |
               |               |                               |                           | |--------------|
               |               |                               |                           |
               |               |                               |                 Collect() |
               |               |                               |<--------------------------|
               |               |                               |                           |
               |               |                      GetAll() |                           |
               |               |<------------------------------|                           |
               |               |                               |                           |
               |               | []string{metrics}             |                           |
               |               |------------------------------>|                           |
               |               |                               |                           |
               |               |                               | concat(metrics)           |
               |               |                               |-------------------------->|
               |               |                               |                           |

```

<details>
 <summary>Code to reproduce diagram</summary>

Build via [text-diagram](http://weidagang.github.io/text-diagram/)

```
object pod_reflector pod_store pod_collector metrics_endpoint

note left of pod_reflector: new pod p1
pod_reflector -> pod_store: Add(p1)
note right of pod_store: generateMetrics(p1)
pod_store -> pod_reflector: nil

note right of metrics_endpoint: GET /metrics
metrics_endpoint -> pod_collector: Collect()
pod_collector -> pod_store: GetAll()
pod_store -> pod_collector: []string{metrics}
pod_collector -> metrics_endpoint: concat(metrics)
```

</details>


## FAQ / Follow up improvements

- If kube-state-metrics only listens on add, update and delete events, how is it
  aware of already existing Kubernetes objects created before kube-state-metrics
  was started? Leveraging Kubernetes client-go, reflectors can initialize all
  existing objects before any add, update or delete events. To ensure no events
  are missed in the long run, periodic resyncs via Kubernetes client-go can be
  triggered. This extra confidence is not a must and should be compared to its
  costs, as Kubernetes client-go already gives decent guarantees on event
  delivery.

- What about metadata (HELP and description) in the /metrics output? As a first
  iteration they would be skipped until we have a better idea on the design.

- How can the cache map be concurrently accessed? The core golang map
  implementation is not thread-safe. As a first iteration a simple mutex should
  be sufficient. Golang's sync.Map might be considered.

- To solve the problem of out of order events send by the Kubernetes API-Server
  to kube-state-metrics, to each blob of time series inside the cache map it can
  keep the Kubernetes resource version. On add and update events, first compare
  the resource version of the event with than the resource version in the cache.
  Only move forward if the former is higher than the latter.

- In case the memory consumption of the time series string blobs is a problem
  the following optimization can be considered: Among the time series strings,
  multiple sub-strings will be heavily duplicated like the metric name. Instead
  of saving unstructured strings inside the cache map, one can structure them,
  using pointers to deduplicate e.g. metric names.

- ...

- Kube-state-metrics does not make use of all properties of all Kubernetes
  objects. Instead of unmarshalling unused properties, their json struct tags or
  their Protobuf representation could be removed.
