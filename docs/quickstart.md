## Usage

This is a guide on how to use the Cluster API Provider for IONOS Cloud (CAPIC) to create a Kubernetes cluster 
on IONOS Cloud. To learn more about the Cluster API, please refer 
to the official [Cluster API book](https://cluster-api.sigs.k8s.io/).

## Dependencies

In order to deploy a K8s cluster with CAPIC, you require the following:

* A machine image, containing pre-installed, matching versions of `kubeadm` and `kubelet`. The machine image can be built with [image-builder](https://github.com/kubernetes-sigs/image-builder) and needs to be available at the
location where the machine will be created on. For more information, [check the custom image guide](custom-image.md).

* `clusterctl`, which you can download from Cluster API (CAPI) [releases](https://github.com/kubernetes-sigs/cluster-api/releases) on GitHub.

* A Kubernetes cluster for running your CAPIC controller.

### Initialize the management cluster

Before creating a Kubernetes cluster on IONOS Cloud, you must initialize a
[management cluster](https://cluster-api.sigs.k8s.io/user/concepts#management-cluster) where CAPI and CAPIC controllers run.

```sh
clusterctl init --infrastructure=ionoscloud
```

### Environment variables

CAPIC requires several environment variables to be set in order to create a Kubernetes cluster on IONOS Cloud.
 They can be exported or saved inside the clusterctl config file at `~/.cluster-api/clusterctl.yaml`

```env
## -- Cloud specific environment variables -- ##
IONOS_TOKEN                                 # The token of the IONOS Cloud account.
IONOS_API_URL                               # The API URL of the IONOS Cloud account.
                                            # Defaults to https://api.ionos.com/cloudapi/v6

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
IONOSCLOUD_MACHINE_SSH_KEYS                 # The SSH keys to be used.
```

### Credential Secret Structure

The `IONOS_TOKEN` should be stored in a secret in the same namespace as the management cluster. 
The secret should have the following structure:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-ionos-credentials
type: Opaque
stringData:
  token: "Token-Goes-Here"
  apiURL: "https://api.ionos.com/cloudapi/v6"
  caBundle: |
    -----BEGIN CERTIFICATE-----
    d293LCBtdWNoIGJhc2U2NCwgc3VjaCBhd2Vzb21lIQ==
    ...
    -----END CERTIFICATE-----
```

Notes:

- The `apiURL` field is optional and defaults to `https://api.ionos.com/cloudapi/v6` if no value was provided.
- The `caBundle` field is optional. It can be used to provide a custom PEM-encoded CA bundle used to validate the
IONOS Cloud API TLS certificate. If unset, the system's root CA set is used. In case of our provided Dockerfile that
would be Debian 12's `ca-certificates` package.

### Create a workload cluster

In order to create a new cluster, you need to generate a cluster manifest with `clusterctl` and then apply it with `kubectl`.

```sh
# Generate a cluster manifest
clusterctl generate cluster ionos-quickstart \
  --infrastructure ionoscloud \
  --kubernetes-version v1.29.2 \
  --control-plane-machine-count 3 \
  --worker-machine-count 3 > cluster.yaml

# Create the workload cluster by applying the manifest
kubectl apply -f cluster.yaml
```

### Check the status of the cluster

```sh 
clusterctl describe cluster ionos-quickstart
```

Wait until the cluster is ready. This can take a few minutes.

### Access the cluster

You can use the following command to get the kubeconfig:

```sh
clusterctl get kubeconfig ionos-quickstart > ionos-quickstart.kubeconfig
```

If you do not have CNI yet, you can use the following command to install a CNI:

```sh
KUBECONFIG=ionos-quickstart.kubeconfig kubectl apply -f https://docs.projectcalico.org/manifests/calico.yaml
```
After that you should see your nodes become ready:

```sh
KUBECONFIG=ionos-quickstart.kubeconfig kubectl get nodes
```

### Installing a CNI

TODO(gfariasalves): Add instructions about installing a CNI or available flavours

### Cleaning a cluster

```sh
kubectl delete cluster ionos-quickstart
```

### Custom Templates

If you need anything specific that requires a more complex setup, we recommend to use custom templates:

```sh
clusterctl generate custom-cluster ionos-quickstart \
  --kubernetes-version v1.27.8 \
  --control-plane-machine-count 1 \
  --worker-machine-count 3 \
  --from ~/workspace/custom-cluster-template.yaml > custom-cluster.yaml
```

### Observability

#### Diagnostics

Access to metrics is secured by default. Before using it, it is necessary to create appropriate roles and role bindings.
For more information, refer to [Cluster API documentation](https://main.cluster-api.sigs.k8s.io/tasks/diagnostics).

### Useful resources

* [Cluster API Book](https://cluster-api.sigs.k8s.io/)
* [Cloud API Docs](https://api.ionos.com/docs/cloud/v6/)
* [IONOS Cloud Docs](https://docs.ionos.com/cloud)
* [IPv6 Documentation](https://docs.ionos.com/cloud/network-services/ipv6)
