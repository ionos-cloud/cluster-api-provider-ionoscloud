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

We create an infrastructure provider. Therefore, we need to follow the naming conventions.

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

Steps:
1. Make sure to have following folder structure: `../ /cluster-api /cluster-api-provider-ionoscloud` (both cluster-api related projects in the same subfolder)
2. Create the tilt settings file `tilt-settings.json` in the `cluster-api` folder:
```json
{
  "default_registry": "ghcr.io/ionos-cloud",
  "provider_repos": [
    "../cluster-api-provider-ionoscloud/"
  ],
  "enable_providers": [
    "ionoscloud",
    "kubeadm-bootstrap",
    "kubeadm-control-plane"
  ],
  "allowed_contexts": [
    "minikube"
  ],
  "kustomize_substitutions": {},
  "extra_args": {
    "ionoscloud": [
      "--v=4"
    ]
  }
}
```
3. Install kind (https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
4. Install tilt   
Linux: https://docs.tilt.dev/install.html#linux   
Mac: https://docs.tilt.dev/install.html#macos
5. Create a kind cluster   
You can create a `kind-config.yaml` file (name doesn't matter) to configure node count and versions:
```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  image: kindest/node:v1.28.0
- role: worker
  image: kindest/node:v1.28.0
```
Choose the version you'd like and add more nodes if needed.   
Now create the kind cluster using the config: `kind create cluster --name <name> --config kind-config.yaml`
6. Now you can run `tilt up`, and everything should run.

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

### Running Tilt
- Create a directory and cd into it
- Clone [cluster-api](https://github.com/kubernetes-sigs/cluster-api)
- Clone [cluster-api-provider-ionoscloud](https://github.com/ionos-cloud/cluster-api-provider-ionoscloud)

- You should now have a directory containing the following git repositories:
```
./cluster-api
./cluster-api-provider-ionoscloud
```

- Change directory to cluster-api: `cd cluster-api`. This directory is the working directory for Tilt.
- Create a file called `tilt-settings.json` with the following contents:

```json
{
  "default_registry": "ghcr.io/ionos-cloud",
  "provider_repos": ["../cluster-api-provider-ionoscloud/"],
  "enable_providers": ["ionoscloud", "kubeadm-bootstrap", "kubeadm-control-plane"],
  "allowed_contexts": ["minikube"],
  "kustomize_substitutions": {},
  "extra_args": {
    "ionoscloud": ["--v=4"]
  }
}
```

This file instructs Tilt to use the cluster-api-provider-ionoscloud. `allowed_contexts` is used to add
allowed clusters other than kind (which is always implicitly enabled).

- If you don't have a cluster, create a new kind cluster:
```
kind create cluster --name capi-test
```
- cluster-api-provider-ionoscloud uses environment variables to connect to the IONOS Cloud API. These need to be set in the shell which spawns Tilt.
  Tilt will pass these to the respective Kubernetes pods created. All variables are documented in `../cluster-api-provider-ionoscloud/envfile.example`.
  Copy `../cluster-api-provider-ionoscloud/envfile.example` to `../cluster-api-provider-ionoscloud/envfile` and make changes pertaining to your configuration.
  For documentation on environment variables, see [usage](Usage.md#environment-variables)

- If you already have a kind cluster with a name that is different to `kind-capi-test`, add this line to `../cluster-api-provider-ionoscloud/envfile`:
```
export CAPI_KIND_CLUSTER_NAME=<yourclustername>
```

- Start Tilt with the following command (with CWD still being ./cluster-api):
```
. ../cluster-api-provider-ionoscloud/envfile && tilt up
```

Press the **space** key to open the Tilt web interface in your browser. Check that all resources turn green and there are no warnings.
You can click on (Tiltfile) to see all the resources.

> **Congratulations** you now have CAPIC running via Tilt. If you make any code changes you should see that CAPIC is automatically rebuilt.
For help deploying your first cluster with CAPIC, see [usage](Usage.md).
