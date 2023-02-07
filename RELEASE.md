# How to cut a new release

## Branch management and versioning strategy

We use [Semantic Versioning](http://semver.org/).

We maintain a separate branch for each minor release, named `release-<major>.<minor>`, e.g. `release-1.1`, `release-2.0`.

The usual flow is to merge new features and changes into the master branch and to merge bug fixes into the latest release branch. Bug fixes are then merged into master from the latest release branch. The master branch should always contain all commits from the latest release branch.

If a bug fix got accidentally merged into master, cherry-pick commits have to be created in the latest release branch, which then have to be merged back into master. Try to avoid that situation.

Maintaining the release branches for older minor releases happens on a best effort basis.

## Prepare your release

* Bump the version in the `VERSION` file in the root of the repository.
* Run `make examples`, which will re-generate all example manifests to use the the new version.
* Make a PR to update:
  * kube-state-metrics image tag for both `quay.io` and `staging-k8s.gcr.io`.
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
* Cut the new release branch, i.e., `release-1.2`, or merge/cherry-pick changes onto the minor release branch you intend to tag the release on
* Cut the new release tag, i.e., `v1.2.0-rc.0`
* Ping Googlers(@loburm/@piosz) to build and push newest image to `k8s.gcr.io` (or to `staging-k8s.gcr.io` in case of release candidates)
* Build and push newest image to `quay.io`(@brancz)

## Stable release

First a release candidate (e.g. `v1.2.0-rc.0`) should be cut. If after a period of 7 days no bugs or issues were reported after publishing the release candidate, a stable release (e.g. `v1.2.0`) can be cut.
