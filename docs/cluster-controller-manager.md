#### Cloud Controller Manager installation

The [Cloud Controller Manager (CCM) | https://kubernetes.io/docs/concepts/architecture/cloud-controller/]  is a
Kubernetes controller that manages the underlying cloud infrastructure. It is responsible for creating and managing
certain cloud resources such as load balancers and also manages certain aspects of the node lifecycle.

If you run the Ionos CCM within your cluster, the
[CCM administration documentation|https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/]
recommends several changes on the cluster. Mainly, several cluster components should be run with the
'--cloud-provider=external' flag. Luckily, with the inons cloud cluster api provider, it is possible to manage this
setting.
Before you generate the cluster manifest, copy the cluster-api-provider template to a new file and alter the
cloud-provider setting in the template.
```shell
cp templates/cluster-template.yaml templates/cluster-template-external.yaml
sed -i 's/cloud-provider: \"\"/cloud-provider: \"external\"/' templates/cluster-template-external.yaml
```

Then use and use the altered template for the generation using the '--from <template_file>' flag.

The ionos cloud CCM can be installed using a [helm | https://helm.sh/] chart. The CCM requires a specific secret,
called ionos-secret, to be present in the namespace in your cluster it is installed to. The helm chart can also manage
the ionos secret for you.

For clusters using a CCM, it is recommended to set the `--cloud-provider=external` flag on the kubelet, kube-. This will

In oder to install the ccm using the helm chart, execute the following steps:
1. Define all required values for the helm chart in a values.yaml file. This file should look like this:
```yaml
---
deployment:
replicas: 3

ccm:
  # This is used in naming resources (IP blocks) managed bu the ccm and naming the metrics. We recommend to use a unique
  # identifier like the cluster name here (optional).
  clusterID:

deployment:
  # The number of replicas to be used for the ccm deployment (defaults to three, #control-plane nodes should be sufficient).
  replicas: 3

# This is only used for full target and generates the ionos-secret with the given values.
# It is intended to be used for installing ccm-ionoscloud to cluster-api based clusters.
# It uses the given token for all secrets. REMEMBER: Ionos tokens have a TTL and must be refreshed.
ionosSecret:
  # The ionos endpoint to be used (required).
  ionosEndpoint: "http://endpoint-ionos-api.mk8s-system.svc.cluster.local/cloudapi/v6"
  # The ionos token to be used this token is used for all DCs (required).
  ionosToken: ""
  # A list of datacenter ids there will be one entry in the secret for each datacenter (required).
  datacenters: []

# This allows you to activate the metrics scraping for the ccm.
metrics:
  # If true, enables metrics scraping.
  enabled: true
  # How often to scrape.
  scrapeInterval: 30s
  # Alternative namespace to create the PodMonitor in, defaults to release namespace.
  namespace: ""
  # Additional labels to add to the PodMonitor.
  additionalLabels: {}
  # Metrics Port
  port: 9100
```
2. Add the ionos cloud ccm registry to your helm registries:
3. Install the helm chart from the ionos cloud ccm registry using the following command:
```shell
helm --kubeconfig <your_cluster_kubeconfig> -n kube-system install ccm-ionoscloud ionoscloud/ccm-ionoscloud --values values.yaml
```
    This will install the ionos cloud ccm to your cluster. It will be installed to the given namespace
    (kube-system in this case). CCM pods will also be scheduled to control planes nodes in your cluster.

## The ionos secret

The ionos-secret is required by the ccm to be able to authnticate with the ionos cloud API.
It has the following structure:
```yaml
apiVersion: v1
kind: Secret
metadata:
    name: ionos-secret
type: Opaque
data:
    <ionos_datacenter_uuid>: <base64 encoded payload>
```

The payload is a json object with the following structure:
```json
{
    "endpoint": <ionos cloud API endpoint>,
    "insecure:" <true if the API endpoint uses a self-signed certificate>,
    "tokens": [<a list of ionos cloud tokens>]
}
```
