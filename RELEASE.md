# How to cut a new release

## Branch management and versioning strategy

We use [Semantic Versioning](http://semver.org/).

We maintain a separate branch for each minor release, named `release-<major>.<minor>`, e.g. `release-1.1`, `release-2.0`.

The usual flow is to merge new features and changes into the main branch and to merge bug fixes into the latest release branch. Bug fixes are then merged into main from the latest release branch. The main branch should always contain all commits from the latest release branch.

If a bug fix got accidentally merged into main, cherry-pick commits have to be created in the latest release branch, which then have to be merged back into main. Try to avoid that situation.

Maintaining the release branches for older minor releases happens on a best effort basis.

## Prepare your release

* Bump the version in the `VERSION` file in the root of the repository.
* Run `make examples`, which will re-generate all example manifests to use the new version.
* Make a PR to update:
  * [Compatibility matrix](README.md#compatibility-matrix)
  * Changelog entry
    * Only include user relevant changes
    * Entries in the [`CHANGELOG.md`](CHANGELOG.md) are meant to be in this order:
    ```
    [CHANGE]
    [FEATURE]
    [ENHANCEMENT]
    [BUGFIX]
    ```
    * All lines should be full sentences
  * kube-state-metrics image tag used in Kubernetes deployment yaml config.
* Cut the new release branch, e.g. `release-1.2`, or merge/cherry-pick changes onto the minor release branch you intend to tag the release on
* Cut the new release tag, e.g. `v1.2.0-rc.0`
* Create a new **pre-release** on github
* New images are automatically built and pushed to `gcr.io/k8s-staging-kube-state-metrics/kube-state-metrics`
* Promote image by sending a PR to [kubernetes/k8s.io](https://github.com/kubernetes/k8s.io) repository. Follow the [example PR](https://github.com/kubernetes/k8s.io/pull/3798). Use [kpromo pr](https://github.com/kubernetes-sigs/promo-tools/blob/main/docs/promotion-pull-requests.md) to update the manifest files in this repository, e.g. `kpromo pr --fork=$YOURNAME -i --project=kube-state-metrics -t=v2.5.0`
* Create a PR to merge the changes of this release back into the main branch.
* Once the PR to promote the image is merged, mark the pre-release as a regular release.

## Stable release

First a release candidate (e.g. `v1.2.0-rc.0`) should be cut. If after a period of 7 days no bugs or issues were reported after publishing the release candidate, a stable release (e.g. `v1.2.0`) can be cut.
