# Developing Cluster API Provider IONOS Cloud

This document describes how to use kind and Tilt for a simplified workflow that offers easy deployments and rapid iterative builds.

## Prerequisites

- Docker: v19.03 or newer
- kind: v0.22.0 or newer
- Tilt: v0.30.8 or newer
- kustomize: provided via make kustomize
- envsubst: provided via make envsubst
- helm: v3.7.1 or newer
- Clone the Cluster API repository locally
- The provider repo

## Getting started

### Create a kind cluster

A script to create a KIND cluster along with a local Docker registry and the correct mounts to run CAPD is included in the cluster-api repo `hack/` folder.

To create a pre-configured cluster run:

```sh
./hack/kind-install-for-capd.sh
```
You can see the status of the cluster with:

```sh
kubectl cluster-info --context kind-capi-test
```

### Create a tilt-settings file

Create the tilt settings file `tilt-settings.yaml` in the `cluster-api` folder:

```yaml
default_registry: ghcr.io/ionos-cloud
provider_repos:
- ../cluster-api-provider-ionoscloud  # path to the provider repo
enable_providers:
- ionoscloud
- kubeadm-bootstrap
- kubeadm-control-plane
allowed_contexts:
- minikube
kustomize_substitutions: {}
extra_args:
- ionoscloud:
  - "--v=4"
```

Note: You're developing the provider, so you might as well want to debug it. For this you might want to add the following to the file above:

```yaml
debug:
  ionoscloud:
    continue: false  # waits for the user to connect to the debugger
    port: 30000  # port where the debugger will be exposed
```

## Create a kind cluster and run Tilt!

To create a pre-configured kind cluster (if you have not already done so) and launch your development environment, you need first of all to copy the [envfile.example](../envfile.example) file in the provider repository and replace the values accordingly. 

Then, run the following in the cluster-api directory:

```sh
. ../cluster-api-provider-ionoscloud/envfile && make tilt-up
```

This will open the command-line HUD as well as a web browser interface. You can monitor Tilt’s status in either location. After a brief amount of time, you should have a running development environment, and you should now be able to create a cluster. There are example worker cluster configs available. These can be customized for your specific needs.

## Notes

This document was adapted from the [Cluster API book](https://cluster-api.sigs.k8s.io/developer/tilt). Please refer to it if you want to use other options with Tilt.