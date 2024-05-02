# Kubernetes Cluster API Provider IONOS Cloud

<p align="center">
<img src="https://raw.githubusercontent.com/kubernetes-sigs/cluster-api/main/docs/book/src/images/introduction.svg"  width="80" style="vertical-align: middle;">
<img src="./docs/LOGO_IONOS_Blue_RGB.png" width="200" style="vertical-align: middle;">
</p>
<p align="center">
<!-- go doc / reference card -->
<a href="https://pkg.go.dev/ionos-cloud/cluster-api-provider-ionoscloud">
<img src="https://godoc.org/ionos-cloud/cluster-api-provider-ionoscloud?status.svg"></a>
<!-- goreportcard badge -->
<a href="https://goreportcard.com/report/ionos-cloud/cluster-api-provider-ionoscloud">
<img src="https://goreportcard.com/badge/ionos-cloud/cluster-api-provider-ionoscloud"></a>
<!-- join kubernetes slack channel for cluster-api-provider-ionos-cloud -->
<!-- <a href="https://kubernetes.slack.com/messages/TBD"> -->
<img src="https://img.shields.io/badge/join%20slack-%23cluster--api--ionoscloud-003d8f?logo=slack"></a>
</p>

Kubernetes-native declarative infrastructure for IONOS Cloud.

## What is the Cluster API Provider IONOS Cloud

[Cluster API][cluster_api] brings declarative, Kubernetes-style APIs to cluster creation, configuration and management.

"This API works with many different cloud services, allowing you to mix and match
different environments easily when using Kubernetes with IONOS Cloud."

## Quick Start

Check out the [Cluster API Quick Start](docs/quickstart.md) to create your first Kubernetes cluster.

<!-- ## Getting Help

If you need help with CAPIC, please visit the [#cluster-api-ionoscloud][slack] channel on Slack or open a [GitHub issue](CONTRIBUTING.md). -->

## Compatibility

### Cluster API Versions (TODO)

### Kubernetes Versions (TODO)

The IONOS Cloud provider is able to install and manage the [versions of Kubernetes supported by the Cluster API (CAPI) project](https://cluster-api.sigs.k8s.io/reference/versions.html#supported-kubernetes-versions).

For more information on Kubernetes version support, see the [Cluster API book](https://cluster-api.sigs.k8s.io/reference/versions.html).

## Documentation

Documentation can be found in the `/docs` directory, and the [index is here](docs/README.md).

## Getting involved and contributing

Are you interested in contributing to cluster-api-provider-ionoscloud? We
maintainers and community would love your suggestions, contributions, and help!
Also, the maintainers can be contacted at any time to learn more about how to get
involved.

To set up for your environment, check out the [development guide](docs/development.md).

In the interest of getting more new people involved, we tag issues with
[`good first issue`][good_first_issue].
These are typically issues that have smaller scope but are good ways to start
to get acquainted with the codebase.

We also encourage ALL active community participants to act as if they are
maintainers, even if you don't have "official" write permissions. This is a
community effort, we are here to serve the Kubernetes community. If you have an
active interest and you want to get involved, you have real power! Don't assume
that the only people who can get things done around here are the "maintainers".

<!-- References -->

<!-- [slack]: https://kubernetes.slack.com/messages/??? -->
[good_first_issue]: https://github.com/ionos-cloud/cluster-api-provider-ionoscloud/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3A%22good+first+issue%22
[bug_report]: https://github.com/ionos-cloud/cluster-api-provider-ionoscloud/issues/new?template=bug_report.md
[feature_request]: https://github.com/kubernetes-sigs/cluster-api-provider-ionoscloud/issues/new?template=feature_request.md
[cluster_api]: https://github.com/ionos-cloud/cluster-api
