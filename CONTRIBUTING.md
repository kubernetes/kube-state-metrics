# Contributing Guidelines

Welcome to Kubernetes. We are excited about the prospect of you joining our [community](https://github.com/kubernetes/community)! The Kubernetes community abides by the CNCF [code of conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md). Here is an excerpt:

_As contributors and maintainers of this project, and in the interest of fostering an open and welcoming community, we pledge to respect all people who contribute through reporting issues, posting feature requests, updating documentation, submitting pull requests or patches, and other activities._

## Getting Started

We have full documentation on how to get started contributing here:

### Semantic Commit Messages

We use [semantic commit messages](https://www.conventionalcommits.org/en/v1.0.0/) in this repository.

They follow this format: `<type>[optional scope]: <description>`

Examples for commit messages following this are:

`feat: allow provided config object to extend other configs`

You can also include a scope within parenthesis:

`fix(scope): Prevent wrong calculation of storage`

Here's a list of types that we use:

| Type | Explanation |
|---|---|
| feat | A new feature |
| fix | A bug fix |
| docs | Documentation only changes |
| style | Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc) |
| refactor | A code change that neither fixes a bug nor adds a feature |
| perf |  A code change that improves performance |
| test | Adding missing tests or correcting existing tests |
| build |Changes that affect the build system or external dependencies (example scopes: gulp, broccoli, npm) |
| ci | Changes to our CI configuration files and scripts (example scopes: Travis, Circle, BrowserStack, SauceLabs) |
| chore | Other changes that don't modify src or test files |
| revert | Reverts a previous commit |

### Local Testing

We recommend you to do local testing on your changes before pushing.

You need to first validate your modules:

```shell
make validate-modules
```

Then, lint check:

```shell
make lint
```

For unit tests:

```shell
make test-unit
```

And for end-to-end integration tests:

```shell
make e2e
```

### Further Information

* [Contributor License Agreement](https://git.k8s.io/community/CLA.md) Kubernetes projects require that you sign a Contributor License Agreement (CLA) before we can accept your pull requests
* [Kubernetes Contributor Guide](http://git.k8s.io/community/contributors/guide) - Main contributor documentation, or you can just jump directly to the [contributing section](http://git.k8s.io/community/contributors/guide#contributing)
* [Contributor Cheat Sheet](https://git.k8s.io/community/contributors/guide/contributor-cheatsheet/README.md) - Common resources for existing developers

## Code Reviews

To keep development moving despite our limited maintainer bandwidth, we are adjusting our review process. We will no longer require a secondary reviewer for maintainer PRs, meaning maintainers can now self-merge their changes after CI passes.

kube-state-metrics runs on community support. If your company depends on this project, this is a great opportunity to get involved! We highly encourage you to help sustain the project by sponsoring maintainer time or allowing your engineers to contribute.

## Mentorship

* [Mentoring Initiatives](https://git.k8s.io/community/mentoring) - We have a diverse set of mentorship programs available that are always looking for volunteers!

## Contact Information

* [Join Slack](http://slack.k8s.io) to sign up and join the Kubernetes Slack. Please make sure to read our [Slack Guidelines](https://github.com/kubernetes/community/blob/master/communication/slack-guidelines.md) before participating.
* The [kube-state-metrics slack channel](https://kubernetes.slack.com/messages/CJJ529RUY) provides an effective communication platform to reach out to members and other users of the project. It offers an alternative to submitting a GitHub issue for when you have questions and issues.
