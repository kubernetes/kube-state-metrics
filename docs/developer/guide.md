# Developer Guide 

This developer guide documentation is intended to assist all contributors in various code contributions.
Any contribution to improving this documentation will be appreciated.

## Table of Contents

- [Add New Kubernetes Resource Metric Collector](#add-new-kubernetes-resource-metric-collector)
- [Squashing Commit History](#squash-commit-history)
### Add New Kubernetes Resource Metric Collector

The following steps are needed to introduce a new resource and its respective resource metrics.

- Reference your new resource(s) to the [docs/README.md](https://github.com/kubernetes/kube-state-metrics/blob/master/docs/README.md#exposed-metrics).
- Reference your new resource(s) in the [docs/cli-arguments.md](https://github.com/kubernetes/kube-state-metrics/blob/master/docs/cli-arguments.md#available-options) as part of the `--resources` flag.
- Create a new `<name-of-resource>.md` in the [docs](https://github.com/kubernetes/kube-state-metrics/tree/master/docs) directory to provide documentation on the resource(s) and metrics you implemented. Follow the formatting of all other resources.
- Add the resource(s) you are representing to the [examples/standard/cluster-role.yaml](https://github.com/kubernetes/kube-state-metrics/blob/master/examples/standard/cluster-role.yaml) under the appropriate `apiGroup` using the `verbs`: `list` and `watch`.
- Reference and add build functions for the new resource(s) in [internal/store/builder.go](https://github.com/kubernetes/kube-state-metrics/blob/master/internal/store/builder.go).
- Reference the new resource in [pkg/options/resource.go](https://github.com/kubernetes/kube-state-metrics/blob/master/pkg/options/resource.go).
- Add a sample Kubernetes manifest to be used by tests in the [tests/manifests/](https://github.com/kubernetes/kube-state-metrics/tree/master/tests/manifests) directory.
- Lastly, and most importantly, actually implement your new resource(s) and its test binary in [internal/store](https://github.com/kubernetes/kube-state-metrics/tree/master/internal/store). Follow the formatting and structure of other resources.

### Squash Commit History

If your PR still has multiple commits after amending previous commits, you must squash multiple commits into a single commit before your PR can be merged. You can check the number of commits on your PR’s Commits tab or by running git log locally. Squashing commits is a form of rebasing.

```
git rebase -i HEAD~<number_of_commits>
```
The -i switch tells git you want to rebase interactively. This enables you to tell git which commits to squash into the first one. For example, you have 3 commits on your branch:

```
12345 commit 4 (2 minutes ago)
6789d commit 3 (30 minutes ago)
456df commit 2 (1 day ago)     
```

You must squash your last three commits into the first one.

```
git rebase -i HEAD~3
```

That command opens an editor with the following:

```
pick 456df commit 2
pick 6789d commit 3
pick 12345 commit 4
```
Change pick to squash on the commits you want to squash, and make sure the one pick commit is at the top of the editor.

```
pick 456df commit 2
squash 6789d commit 3
squash 12345 commit 4
```
Save and close your editor. Then push your squashed commit with git push --force-with-lease origin <branch_name>.
