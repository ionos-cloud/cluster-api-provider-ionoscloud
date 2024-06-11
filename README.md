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
<!-- sonarcloud badge -->
<a href="https://sonarcloud.io/summary/new_code?id=ionos-cloud_cluster-api-provider-ionoscloud">
<img src="https://sonarcloud.io/api/project_badges/measure?project=ionos-cloud_cluster-api-provider-ionoscloud&metric=alert_status&token=61ea2f753f2b2a3ed9a2cf966248fdd57d7f6ebd" alt="Quality Gate Status"></a>
<!-- join kubernetes slack channel for cluster-api-provider-ionos-cloud -->
<!-- <a href="https://kubernetes.slack.com/messages/TBD"> -->
<!-- <img src="https://img.shields.io/badge/join%20slack-%23cluster--api--ionoscloud-003d8f?logo=slack"></a> -->
</p>

Kubernetes-native declarative infrastructure for IONOS Cloud.

## What is the Cluster API Provider IONOS Cloud

The Cluster API Provider IONOS Cloud makes declarative provisioning of multiple Kubernetes clusters through Cluster API on IONOS Cloud infrastructure possible.

[Cluster API][cluster_api] is a Kubernetes sub-project focused on providing declarative APIs and tooling to simplify provisioning, upgrading, and operating multiple Kubernetes clusters.

Started by the Kubernetes Special Interest Group (SIG) Cluster Lifecycle, the Cluster API project uses Kubernetes-style APIs and patterns to automate cluster lifecycle management for platform operators. The supporting infrastructure, like virtual machines, networks, load balancers, and VPCs, as well as the Kubernetes cluster configuration are all defined in the same way that application developers operate deploying and managing their workloads. This enables consistent and repeatable cluster deployments across a wide variety of infrastructure environments.

## Quick Start

Check out the [Cluster API Quick Start](docs/quickstart.md) to create your first Kubernetes cluster.

<!-- ## Getting Help

If you need help with CAPIC, please visit the [#cluster-api-ionoscloud][slack] channel on Slack or open a [GitHub issue](CONTRIBUTING.md). -->

## Compatibility

### Cluster API Versions

This provider's versions are compatible with the following versions of Cluster API:

|                        | Cluster API v1beta1 (v1.7) |
|------------------------|:--------------------------:|
| CAPIC v1alpha1 (v0.2)  |             ✓              |

### Kubernetes Versions 

The IONOS Cloud provider is able to install and manage the [versions of Kubernetes supported by the Cluster API (CAPI) project](https://cluster-api.sigs.k8s.io/reference/versions.html#supported-kubernetes-versions).

For more information on Kubernetes version support, see the [Cluster API book](https://cluster-api.sigs.k8s.io/reference/versions.html).

## Documentation

Documentation can be found in the `/docs` directory, and the [index is here](docs/).

## Getting involved and contributing

Are you interested in contributing to cluster-api-provider-ionoscloud? We, the
maintainers and the community, would love your suggestions, contributions, and help!
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
[cluster_api]: https://github.com/kubernetes-sigs/cluster-api
