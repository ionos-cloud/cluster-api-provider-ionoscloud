# What is IPAM?
IPAM (IP Address Management) is a system used to manage IP address allocation and tracking within a network. In Kubernetes, IPAM is crucial for managing IP addresses across dynamic and often ephemeral workloads, ensuring each network interface within the cluster is assigned a unique and valid IP address.

## Why Use IPAM?
- **Automation**: Simplifies network configuration by automating IP assignment.
- **Scalability**: Supports dynamic scaling of clusters by efficiently managing IP addresses.
- **Flexibility**: Works with various network topologies and can integrate with both cloud-based and on-premises IPAM solutions.
- **Security**: Reduces the risk of IP conflicts and unauthorized access by ensuring each node and pod has a unique IP address.

## Prerequisites for Using IPAM in Kubernetes
To use IPAM, you need an IPAM provider. One such provider is the [Cluster API IPAM Provider In-Cluster](https://github.com/kubernetes-sigs/cluster-api-ipam-provider-in-cluster). This provider allows Kubernetes to integrate IPAM functionalities directly into its cluster management workflow.

## Setting Up an IPAM Provider
- **Install the IPAM Provider**: Deploy the IPAM provider in your Kubernetes cluster. This will typically involve deploying custom controllers and CRDs (Custom Resource Definitions) to manage IP pools.
- **Create IP Pools**: Define IP pools that will be used for assigning IPs to network interfaces. These can be specific to a cluster (InClusterIPPool) or shared across clusters (GlobalInClusterIPPool).

## Using IPAM with IonosCloudMachine
### Example YAML Configuration
Let's explore how to integrate IPAM with your IonosCloudMachine resource in Kubernetes.
```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: IonosCloudMachine
metadata:
  name: example-machine
spec:
  ipv4PoolRef:
    apiGroup: ipam.cluster.x-k8s.io
    kind: InClusterIPPool
    name: primary-node-ips
  additionalNetworks:
  - networkID: 3
    ipv4PoolRef:
      apiGroup: ipam.cluster.x-k8s.io
      kind: InClusterIPPool
      name: additional-node-ips
```
 ### Explanation of Configuration
  
```yaml
ipv4PoolRef:
  apiGroup: ipam.cluster.x-k8s.io
  kind: InClusterIPPool
  name: primary-node-ips
```
- **apiGroup**: Specifies the API group for the IPAM configuration, ensuring the correct API resources are targeted.
- **kind**: The type of IP pool being referenced. In this case, it's InClusterIPPool, which is specific to the current cluster.
- **name**: The name of the IP pool (primary-node-ips) from which the primary NIC will obtain its IP address.
