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


TODO(lubedacht): Add proper cluster-api development setup guide using Tilt.

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -k config/samples/
```

2. Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/cluster-api-provider-ionoscloud:tag
```

3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/cluster-api-provider-ionoscloud:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller from the cluster:

```sh
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)
