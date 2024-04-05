## Usage

---

This is a guide on how to use the Cluster API Provider for IONOS Cloud (CAPIC) to create a Kubernetes cluster 
on IONOS Cloud. To learn more about the Cluster API, please refer 
to the official [Cluster API book](https://cluster-api.sigs.k8s.io/).

## Table of Contents

---

* [Usage](#usage)
  * [Prerequisites](#prerequisites)
  * [Quickstart](#quickstart)
    * [Case 1: Using a local provider](#case-1-using-a-local-provider)
    * [Case 2: The provider is already available in clusterctl](#case-2-the-provider-is-already-available-in-clusterctl)
    * [Configuring the management cluster](#configuring-the-management-cluster)
  * [Environment variables](#environment-variables)
    * [Create a workload cluster](#create-a-workload-cluster)
  * [Next Steps](#next-steps)
  * [Troubleshooting](#troubleshooting)

## Prerequisites

---

Before you can use CAPIC, you need to have the following prerequisites:

* A Kubernetes cluster which can run the required providers for CAPIC.
* An image, which is used to create the Kubernetes cluster. This image has to be available in the IONOS Cloud location
  of the datacenter where you want to create the Kubernetes cluster.
  * The image can be built via [image-builder](https://github.com/kubernetes-sigs/image-builder)
    * It must be built as a raw QEMU image. Refer to the [custom image](./custom-image.md) documentation for more information. 
* `clusterctl`, which can be installed via the [official documentation](https://cluster-api.sigs.k8s.io/user/quick-start.html#install-clusterctl).
* A datacenter in IONOS Cloud where you want to create the Kubernetes cluster.

## Quickstart

---

In order to install Cluster API Provider for IONOS Cloud (CAPIC), you need to have a Kubernetes cluster up and running,
and `clusterctl` installed.

### Case 1: Using a local provider

---

If the provider is not yet added to the list of providers in `clusterctl`, you can bootstrap the management cluster
using a local provider. Refer to [local provider](./local-provider.md) for more information.

### Case 2: The provider is already available in clusterctl

---

In this case you can simply follow the steps below. Make sure you are using a version of `clusterctl` which
supports the `IONOS Cloud provider`.

### Configuring the management cluster

---

Before you can create a Kubernetes cluster on IONOS Cloud, you need to configure the management cluster.
Currently, the controller has no need of any special configuration, so you can just run the following command:

```sh
clusterctl init --infrastructure=ionoscloud
```


### Environment variables

---

CAPIC requires several environment variables to be set in order to create a Kubernetes cluster on IONOS Cloud.

```env
## -- Cloud specific environment variables -- ##
IONOS_TOKEN                                 # The token of the IONOS Cloud account.
IONOS_API_URL                               # The API URL of the IONOS Cloud account.
                                            #   Defaults to https://api.ionos.com/cloudapi/v6

## -- Cluster API related environment variables -- ##
CONTROL_PLANE_ENDPOINT_IP                   # The IP address of the control plane endpoint.        
CONTROL_PLANE_ENDPOINT_PORT                 # The port of the control plane endpoint.
CONTROL_PLANE_ENDPOINT_LOCATION             # The location of the control plane endpoint.
CLUSTER_NAME                                # The name of the cluster.
KUBERNETES_VERSION                          # The version of Kubernetes to be installed (can also be set via clusterctl).

## -- Kubernetes Cluster related environment variables -- ##
IONOSCLOUD_CONTRACT_NUMBER                  # The contract number of the IONOS Cloud contract.
IONOSCLOUD_DATACENTER_ID                    # The datacenter ID where the cluster should be created.
IONOSCLOUD_MACHINE_NUM_CORES                # The number of cores.
IONOSCLOUD_MACHINE_MEMORY_MB                # The memory in MB.
IONOSCLOUD_MACHINE_IMAGE_ID                 # The image ID.
IONOSCLOUD_MACHINE_CPU_FAMILY               # The CPU family.
IONOSCLOUD_MACHINE_SSH_KEYS                 # The SSH keys to be used.
```

### Credential Secret Structure

---

The `IONOS_TOKEN` should be stored in a secret in the same namespace as the management cluster. 
The secret should have the following structure:

The `apiURl` field is optional and defaults to `https://api.ionos.com/cloudapi/v6` if no value was provided.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-ionos-credentials
type: Opaque
stringData:
  token: "Token-Goes-Here"
  apiURL: "https://api.ionos.com/cloudapi/v6"
```

### Create a workload cluster

---

In order to create a new cluster, you need to generate a cluster manifest.

```sh
# Make sure you have the required environment variables set
$ source envfile
# Generate a cluster manifest
$ clusterctl generate cluster ionos-quickstart \
  --infrastructure ionoscloud \
  --kubernetes-version v1.29.2 \
  --control-plane-machine-count 3 \
  --worker-machine-count 3 > cluster.yaml

# Create the workload cluster by applying the manifest
$ kubectl apply -f cluster.yaml
```

### Next Steps

---

TODO

### Troubleshooting

---

TODO
