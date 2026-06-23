# ResourceClaim Metrics

| Metric name                                                   | Metric type | Description | Labels/tags | Status |
| ------------------------------------------------------------- | ----------- | ----------- | ----------- | ------ |
| kube_resourceclaim_info                                       | Gauge       | Information about resource claim. | `resourceclaim`=&lt;resourceclaim-name&gt; <br> `namespace`=&lt;resourceclaim-namespace&gt; | EXPERIMENTAL |
| kube_resourceclaim_created                                    | Gauge       | Unix creation timestamp | `resourceclaim`=&lt;resourceclaim-name&gt; <br> `namespace`=&lt;resourceclaim-namespace&gt; | EXPERIMENTAL |
| kube_resourceclaim_status_allocated                           | Gauge       | Indicates whether the resource claim has been allocated. | `resourceclaim`=&lt;resourceclaim-name&gt; <br> `namespace`=&lt;resourceclaim-namespace&gt; | EXPERIMENTAL |
| kube_resourceclaim_status_reserved_for                        | Gauge       | Indicates which consumers have currently reserved the resource claim. | `resourceclaim`=&lt;resourceclaim-name&gt; <br> `namespace`=&lt;resourceclaim-namespace&gt; <br> `consumer_apigroup`=&lt;consumer-apigroup&gt; <br> `consumer_resource`=&lt;consumer-resource&gt; <br> `consumer_name`=&lt;consumer-name&gt; <br> `consumer_uid`=&lt;consumer-uid&gt; | EXPERIMENTAL |
| kube_resourceclaim_allocation_device_info                     | Gauge       | Allocation information about the devices allocated to the resource claim. | `resourceclaim`=&lt;resourceclaim-name&gt; <br> `namespace`=&lt;resourceclaim-namespace&gt; <br> `request`=&lt;request-name&gt; <br> `driver`=&lt;driver-name&gt; <br> `pool`=&lt;pool-name&gt; <br> `device`=&lt;device-name&gt; | EXPERIMENTAL |
