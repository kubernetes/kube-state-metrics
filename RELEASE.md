## Procedures
* make a PR to update:
  * kube-state-metrics image tag for both `quay.io` and `k8s.gcr.io`
  * compatibility matrix
  * change log
  * kube-state-metrics image tag used in Kubernetes deployment yaml config
* cut the new release branch, i.e., `release-1.2`
* cut the new release tag, i.e., `v1.2.0-rc.0`
* ping Googlers(currently @loburm/@piosz) to build and push newest image to `k8s.gcr.io`
* build and push newest image to `quay.io`(currently @brancz)
