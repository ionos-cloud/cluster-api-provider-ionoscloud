# Kubernetes Cluster API Provider for IONOS Cloud - CAPIC

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ionos-cloud_cluster-api-provider-ionoscloud&metric=alert_status&token=61ea2f753f2b2a3ed9a2cf966248fdd57d7f6ebd)](https://sonarcloud.io/summary/new_code?id=ionos-cloud_cluster-api-provider-ionoscloud)

## Table of Contents

---

- [Overview](#overview)
- [Documentation](#documentation)
- [Launching a Kubernetes cluster on IONOS Cloud](#launching-a-kubernetes-cluster-on-ionos-cloud)
- [Features](#features)
- [Maintainers](#maintainers)
- [License](#license)
<!-- TODO -[Contributing](./CONTRIBUTING.md) -->

## Overview

---

The [Cluster API](https://github.com/kubernetes-sigs/cluster-api) brings declarative, Kubernetes-style APIs to cluster creation, configuration and management.

## Documentation

---

Documentation can be found in the ./docs folder. 

To get started with developing, please see [our development docs](./docs/Development.md)

## Launching a Kubernetes cluster on IONOS Cloud

---

Check out the [quickstart guide](./docs/quickstart.md) to get started with launching a cluster on IONOS Cloud.

## Features

---

* Native Kubernetes manifests and API.
* Manages the bootstrapping of LANs, Failover Groups and VMs on IONOS Cloud.
* Deploys Kubernetes control planes into provided virtual data center in IONOS Cloud.
* Doesn't use SSH for bootstrapping nodes.
* Installs only the minimal components to bootstrap a control plane and workers.
* Uses IPv6 by default.

## Maintainers

| Username              |
|-----------------------|
| @piepmatz             |
| @gfariasalves-ionos   |
| @lubedacht            |
| @wikkyk               |


## License

Copyright 2024 IONOS Cloud.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

