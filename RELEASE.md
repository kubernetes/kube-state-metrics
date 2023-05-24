# How to cut a new release

## Branch management and versioning strategy

### Semantic versioning

We use [Semantic Versioning](http://semver.org/), and thus, maintain a separate branch for each minor release, named 
`release-<major>.<minor>`, e.g. `release-1.1`, `release-2.0`.

### Workflow

- The usual flow is to merge new features and changes into the main branch and to merge (the later) bug fixes into the 
latest release branch. 
- Bug fixes are then merged into main from the latest release branch. 
- The main branch should always contain **all** commits from the latest release branch.
- If a bug fix got accidentally merged into main, cherry-pick commits have to be created in the latest release branch, 
which then have to be merged back into main. Try to avoid that situation.
- Maintaining the release branches for older minor releases happens on a **best effort** basis.

## Preparing the release

Note: For a stable release, first a release candidate (e.g. `v1.2.0-rc.0`) should be cut. If, after a period of 
**7 days**, no bugs or issues were reported after publishing the release candidate, a stable release (e.g. `v1.2.0`) 
can be cut.

### Workflow

* Bump the version in [`VERSION.md`](VERSION.md).
* Run `make examples`, which will re-generate all example manifests to use the newer version.
* Cut the new release branch, e.g. `release-1.2`, or merge/cherry-pick changes onto the minor release branch you intend
to tag the release on.
* Include the following changes in the release PR (cut from `main`).
  * Update the [compatibility matrix](README.md#compatibility-matrix).
  * Add changelog entries.
    * Entries in the [`CHANGELOG.md`](CHANGELOG.md) are meant to be in this order:
    ```
    [BUGFIX]
    [CHANGE]
    [ENHANCEMENT]
    [FEATURE]
    ```
    * To generate changelog, follow the process below:
      * Fetch all commits since last release, for instance, when preparing for `v2.9.0`, do 
      `git log --oneline --decorate upstream/release-2.8..upstream/main`.
      * Remove all merge commits, `fixup!`s (if any), and any commits that might have crept in from a PR, other
        than it's original commit, unless wanted otherwise.
      * Remove all non-user-facing changes.
      * Label the remaining commits under the aforementioned categories, and futher condense commit groups
        under their associated PRs, wherever possible.
      * Format the message in the manner: `[<category>] <PR header> <PR number> <Author>`.
    * All lines should be full sentences.
* Cut the new release-candidate tag, e.g. `v1.2.0-rc.0`.
* Create a new **pre-release**.
* Create a PR to merge the changes of this release back into the `main` branch.
* Once the PR to promote the image is merged, mark the pre-release as a regular release. This must be done after the 
  **7-day** period is over.

### Post-release chores

* Promote the latest release image, refer [k8s.io/#4750](https://github.com/kubernetes/k8s.io/pull/3798) for more 
  details. Also see [promotion-pull-requests.md#promoting-images](https://github.com/kubernetes-sigs/promo-tools/blob/main/docs/promotion-pull-requests.md#promoting-images).
  * For instance, `kpromo pr --fork mrueg -i --project kube-state-metrics --reviewers="@fpetkovski,@dgrisonnet,@rexagod" -t=v2.8.2 --image="" --digests=""`.
  * Open a PR promoting the image (based on aforementioned links).
  * New images will be automatically built and pushed to `gcr.io/k8s-staging-kube-state-metrics/kube-state-metrics`.

