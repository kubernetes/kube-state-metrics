## Procedures
* make a PR to update:
  * kube-state-metrics image tag for both `quay.io` and `staging-k8s.gcr.io`
  * compatibility matrix
  * changelog entry
    * only include user relevant changes
    * entries should follow in order below
    ```
    [CHANGE]
    [FEATURE]
    [ENHANCEMENT]
    [BUGFIX]
    ```
    * all lines should be full sentences
  * kube-state-metrics image tag used in Kubernetes deployment yaml config
* cut the new release branch, i.e., `release-1.2`, or merge/cherry-pick changes onto the minor release branch you intend to tag the release on
* cut the new release tag, i.e., `v1.2.0-rc.0`
* ping Googlers(@loburm/@piosz) to build and push newest image to `staging-k8s.gcr.io`
* build and push newest image to `quay.io`(@brancz)
