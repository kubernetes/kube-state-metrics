# Maintaining kube-state-metrics

kube-state-metrics is welcoming contributions from the community. If you are interested in intensifying your contributions and becoming a maintainer, this doc describes the necessary steps.

As part of the Kubernetes project, we use the community membership process as described [here](https://github.com/kubernetes/community/blob/master/community-membership.md). We do not adhere strictly to the numbers of contributions and reviews. Still as becoming a maintainer is a trust-based process and we desire positive outcomes for the project, we look for a long-term interest and engagement.

## Adding a new reviewer

* Ensure the new reviewer is a member of the [kubernetes organization](https://github.com/kubernetes/org/blob/main/config/kubernetes/org.yaml).
* Add the new reviewer to the [OWNERS](OWNERS) file to be able to review pull requests.
* Add the new reviewer to the [kube-state-metrics-maintainers group](https://github.com/kubernetes/org/blob/main/config/kubernetes/sig-instrumentation/teams.yaml), to gain write access to the kube-state-metrics repository (e.g. for creating new releases).

## Adding a new approver

* Ensure the new approver is already a reviewer in the [OWNERS](OWNERS) file.
* Add the new approver to the [OWNERS](OWNERS) file to be able to approve pull requests.
* Add the new approver to the [SECURITY_CONTACTS](SECURITY_CONTACTS) file to be able to get notified on security related incidents.
* Add the new approver to the [kube-state-metrics-admin group](https://github.com/kubernetes/org/blob/main/config/kubernetes/sig-instrumentation/teams.yaml), to get admin access to the kube-state-metrics repository.
* Add the new approver to the k8s.io [OWNERS](https://github.com/kubernetes/k8s.io/blob/main/k8s.gcr.io/images/k8s-staging-kube-state-metrics/OWNERS) file to be able to approve image promotion from the staging registry.
