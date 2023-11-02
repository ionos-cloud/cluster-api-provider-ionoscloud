### Temporary development documentation

## Note

This document contains helpful development information and hacks,
which should help you to get started with the 

## Scaffolding

Brief summary of commands, which were used to scaffold the project

### Initialization

Init the project

```bash
kubebuilder init \
--domain cluster.x-k8s.io \
--repo github.com/ionos-cloud/cluster-api-provider-ionoscloud \
--project-name cluster-api-provider-ionoscloud
```

### Create API types

We create an infrastructure provider. Therefore we need to follow the naming conventions.

[Resource Naming](https://cluster-api.sigs.k8s.io/developer/providers/implementers-guide/naming.html?highlight=cluster.x-k8s.io#resource-naming)

Our group would be `infrastructure` to get `infrastructure.cluster.x-k8s.io` as group.
Initial version will be v1alpha1

```bash
# Create the cluster resource 

kubebuilder create api \
--resource \
--controller \
--group infrastructure \
--version v1alpha1 \
--kind IonosCloudCluster \
--namespaced

# Create the machine resource

kubebuilder create api \
--resource \
--controller \
--group infrastructure \
--version v1alpha1 \
--kind IonosCloudMachine \
--namespaced

```

### Setup local test environment

TODO: convert steps to proper documentation

Steps:
1. make sure to have folder structure
../
/cluster-api
/cluster-api-provider-ionoscloud
2. tilt settings file
3. install kind
4. install tilt
5. create kind cluster
6. tilt up

### Make sure our api resources implement the contracts

[Cluster Contract](https://cluster-api.sigs.k8s.io/developer/architecture/controllers/cluster#infrastructure-provider)

[Machine Contract](https://cluster-api.sigs.k8s.io/developer/architecture/controllers/machine#infrastructure-provider)
