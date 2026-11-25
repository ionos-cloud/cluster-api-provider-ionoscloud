# Flatcar Container Linux

CAPIC can run nodes on [Flatcar Container Linux](https://www.flatcar.org/). Flatcar is configured by
[Ignition](https://coreos.github.io/ignition/) instead of cloud-init, so the kubeadm bootstrap provider (CABPK) must
render the bootstrap data with `format: ignition`.

The reasoning behind the design, what was tested on IONOS Cloud, and how to build the images are in the
[design document](flatcar-ignition-design.md).

## How it works

IONOS Cloud has no metadata service. It writes the `userData` of a volume into that volume before the server starts,
if the volume's image supports it. CAPIC uses this to deliver the Ignition config:

1. For a Machine whose bootstrap secret has `format: ignition`, CAPIC creates the server with two volumes:
   - `vol-<machine>`: the boot volume from `spec.disk.image`, without `userData`, `bootOrder: PRIMARY`.
   - `cfg-<IonosCloudMachine UID>`: the config volume from `spec.bootstrapVolume.image`, with the Ignition config as `userData`,
     `bootOrder: NONE`. Its size is the size of the image.
2. IONOS writes the config to `config/user_data` on the config volume's `ionos-ignition` partition.
3. The Flatcar image has the kernel arguments `ignition.config.device=ionos-ignition` and
   `ignition.config.path=config/user_data`, so Ignition reads the config from there on the first boot.
4. Once the Machine has a node reference, CAPIC deletes the config volume. IONOS hot-unplugs it from the running
   server. Ignition only runs on the first boot, so the node never needs the volume again. The volume holds the
   bootstrap data, which includes the cluster CA keys on control plane nodes, so it must not outlive the join.
5. When a Machine is deleted before that, for example because the node never joined, deleting the server detaches
   the config volume, and CAPIC deletes leftover volumes named `cfg-<IonosCloudMachine UID>`.

If the bootstrap data is in Ignition format and `spec.bootstrapVolume` is not set, CAPIC does not create the server and
sets the `MachineProvisioned` condition to `False` with the reason `BootstrapVolumeRequired`.

Machines with `format: cloud-config` (the default) are not affected; `spec.bootstrapVolume` is ignored for them.

A Flatcar node cannot do its first boot without the config volume: Ignition waits about 30 seconds for the
`ionos-ignition` device and then fails the boot. The Machine never gets a node reference. Use a MachineHealthCheck
with `nodeStartupTimeout` to replace such Machines.

## Enable Ignition in CABPK

Ignition support is an alpha feature of CABPK. Enable it before `clusterctl init`:

```sh
export EXP_KUBEADM_BOOTSTRAP_FORMAT_IGNITION=true
clusterctl init --infrastructure ionoscloud
```

This sets the `KubeadmBootstrapFormatIgnition` feature gate on both the kubeadm bootstrap (CABPK) and the kubeadm
control plane (KCP) controller. On an existing management cluster, set `KubeadmBootstrapFormatIgnition=true` in the
existing `--feature-gates` argument of both deployments (it contains `KubeadmBootstrapFormatIgnition=false` by
default). Do not add a second `--feature-gates` argument: it replaces the first one.

## Images

Both images must exist in every location you use. The recipe to build them is in the
[design document](flatcar-ignition-design.md#building-the-images-yourself).

The config volume image must be a private image: IONOS requires a password or an SSH key for volumes from public
images, CAPIC sets neither, and IONOS rejects the request.

| Image | Content | Required properties |
|---|---|---|
| Node image | Flatcar 4820 or newer (Ignition 2.27 or newer). OEM `grub.cfg` with `oem_id=qemu` and the two `ignition.config.*` kernel arguments. Depending on the flavor: the Kubernetes sysext and the `ionos-cloud` sysext. | `licenceType: LINUX`, `discVirtioHotPlug: true`, `nicHotPlug: true` |
| Config volume image | A GPT disk with one ext4 partition labelled `ionos-ignition` that holds only the marker file `USER_DATA_INJECTION_FLATCAR`. No cluster data. | `cloudInit: V1` (otherwise IONOS rejects `userData`), `discVirtioHotUnplug: true` (for the removal after the join) |

The node images built by IONOS also mask `update-engine.service`, `locksmithd.service`, `systemd-sysupdate.timer`
and `systemd-sysupdate-reboot.timer`. The templates mask them as well, so nodes never update or reboot themselves.
Update the OS by rolling out Machines with a new image.

If you select images with `spec.disk.image.selector` or `spec.bootstrapVolume.image.selector`: the config volume image
name has no Kubernetes version, and CAPIC ignores `useMachineVersion` for `spec.bootstrapVolume.image`. The CRD still
shows the default `true` for it.

## The ionos-cloud sysext

Flatcar's `coreos-metadata.service` (Afterburn) has no IONOS Cloud provider. The `ionos-cloud`
[systemd-sysext](https://www.flatcar.org/docs/latest/provisioning/sysext/) adds a `coreos-metadata.service` drop-in
that writes `/run/metadata/flatcar`:

| Variable | Source |
|---|---|
| `COREOS_IONOSCLOUD_HOSTNAME` | Kernel host name, set from DHCP. IONOS sends the server name, which is the `IonosCloudMachine` name. |
| `COREOS_IONOSCLOUD_INSTANCE_ID` | `/sys/class/dmi/id/product_uuid`, which is the IONOS server ID. |
| `COREOS_IONOSCLOUD_IPV4_DEFAULT` | Source address of the default route. |
| `COREOS_IONOSCLOUD_IPV4_PRIVATE` | First RFC 1918 or RFC 6598 address. |
| `COREOS_IONOSCLOUD_IPV4_PUBLIC` | First other global address. |

These names are a contract: variables are only added, never renamed or removed.

The sysext also contains `ionos-sysext-fetch.service`, which pulls sysexts pinned by digest from an OCI registry if
`/etc/ionos-cloud/sysext.d/<name>.conf` files exist. If kubeadm needs a sysext pulled this way, add
`ionos-sysext-fetch.service` to `Requires=` and `After=` of the `kubeadm.service` drop-in, so kubeadm does not run
when the fetch fails.

Install the sysext as `/opt/extensions/ionos-cloud/<file>.raw` with the symlink `/etc/extensions/ionos-cloud.raw`
pointing to it. The name in the extension-release file is `ionos-cloud`, and systemd-sysext only merges the image if
its file name matches.

The source is in [`hack/flatcar/ionos-cloud-sysext`](../hack/flatcar/ionos-cloud-sysext). Build it on Linux with
`squashfs-tools` installed:

```sh
hack/flatcar/ionos-cloud-sysext/build.sh out/
# -> out/ionos-cloud-<version>-x86-64.raw, prints its sha512
```

## Flavors

None of the flavors installs a CNI.

| Flavor | Node image | Node metadata |
|---|---|---|
| `flatcar` | Kubernetes sysext and `ionos-cloud` sysext baked in | `coreos-metadata.service` from the `ionos-cloud` sysext |
| `flatcar-without-ionos-sysext` | Kubernetes sysext baked in, no `ionos-cloud` sysext (the image from the recipe in the design document) | A script in the template computes the same variables |
| `flatcar-sysext` | Generic: no Kubernetes, `ionos-cloud` sysext baked in | `coreos-metadata.service` from the `ionos-cloud` sysext. Ignition downloads the Kubernetes sysext from the [Flatcar sysext-bakery](https://github.com/flatcar/sysext-bakery) and verifies its sha512. |

All flavors:

- set `format: ignition` on the `KubeadmControlPlane` and the `KubeadmConfigTemplate`;
- run kubeadm after `containerd.service` and the metadata step, and fill in the kubeadm config with `envsubst`;
- register the node with the server name, `node-ip` set to the source address of the default route, and
  `provider-id: ionos://<server ID>`. No CCM is needed for the Machine to get its node reference;
- add the SSH keys to the user `core`;
- run kube-vip as a static pod on the control plane, as the default flavor does;
- stop before kubeadm runs if the server ID is empty, instead of registering the node with an invalid provider ID;
- run the scripts they write to `/etc` with `bash <script>`: Flatcar 4820 mounts `/etc` as an overlay with `noexec`.

Unlike the default flavor, the Flatcar flavors do not set NTP servers, sysctls, the `ip_vs` kernel modules or the sshd
hardening. IONOS DHCP sends no NTP server, so `systemd-timesyncd` uses its default pool, which needs egress to the
internet.

The variables are the ones of the default flavor (see the [quickstart](quickstart.md#environment-variables)), plus:

| Variable | Flavors | Description |
|---|---|---|
| `IONOSCLOUD_MACHINE_IMAGE_ID` | all | The Flatcar node image. |
| `IONOSCLOUD_BOOTSTRAP_VOLUME_IMAGE_ID` | all | The config volume image. |
| `KUBERNETES_SYSEXT_SHA512` | `flatcar-sysext` | sha512 of `kubernetes-${KUBERNETES_VERSION}-x86-64.raw`. |
| `FLATCAR_SYSEXT_BASE_URL` | `flatcar-sysext` | Optional. Where to download the Kubernetes sysext from. Defaults to `https://extensions.flatcar.org/extensions`. |

The sysext-bakery publishes only sha256 sums, and CABPK's Ignition output (spec 2.3) only supports sha512, so compute
the sum yourself and compare the sha256 with the bakery's `SHA256SUMS`:

```sh
K=v1.37.1
curl -fLO https://extensions.flatcar.org/extensions/kubernetes-$K-x86-64.raw
curl -fsSL https://extensions.flatcar.org/extensions/kubernetes/SHA256SUMS | grep " kubernetes-$K-x86-64.raw$" | sha256sum -c
export KUBERNETES_SYSEXT_SHA512=$(sha512sum kubernetes-$K-x86-64.raw | cut -d' ' -f1)
```

For `v1.37.1` the sha512 is
`e2cfbe5c4edbe917d41cfe59575217a276c717932874b35e9e03b77f5b563586030f1491d9b2758b7b2f456d474805da707ef92fd04a70cea9a0ebf4ea9879c9`.

In the `flatcar-sysext` flavor the Kubernetes version is part of the bootstrap configuration. To upgrade, update the
sysext URL, link and sha512 in the `KubeadmControlPlane` together with `spec.version`, and create a new
`KubeadmConfigTemplate` for the MachineDeployments.

Create a cluster:

```sh
clusterctl generate cluster flatcar-1 \
  --infrastructure ionoscloud \
  --kubernetes-version v1.37.1 \
  --control-plane-machine-count 1 \
  --worker-machine-count 2 \
  --flavor flatcar > cluster.yaml

kubectl apply -f cluster.yaml
```

## CNI

The nodes stay `NotReady` until a CNI is installed. The Kubernetes sysext puts the reference CNI plugins into
`/opt/cni/bin`, and `/etc/cni/net.d` is writable, so the usual CNI manifests and charts work. The templates use the pod
CIDR `192.168.0.0/16`.

Example with [Cilium](https://docs.cilium.io/):

```sh
clusterctl get kubeconfig flatcar-1 > flatcar-1.kubeconfig
helm repo add cilium https://helm.cilium.io/
helm install cilium cilium/cilium --kubeconfig flatcar-1.kubeconfig \
  --namespace kube-system \
  --set ipam.mode=kubernetes
```

`ipam.mode=kubernetes` makes Cilium use the pod CIDRs that Kubernetes assigns to the nodes from the cluster's pod
CIDR.

## Troubleshooting

Log in as `core`. Useful places:

- `journalctl -u kubeadm.service`: the output of `/etc/kubeadm.sh` (pre-kubeadm commands, `kubeadm init/join`).
- `journalctl -u coreos-metadata.service` and `/run/metadata/flatcar` (flavors with the `ionos-cloud` sysext), or
  `/run/metadata/ionos-cloud` (`flatcar-without-ionos-sysext`).
- `journalctl -b -t ignition`: Ignition, for example when it did not find the config volume.
- `systemd-sysext status`: the merged sysexts.
