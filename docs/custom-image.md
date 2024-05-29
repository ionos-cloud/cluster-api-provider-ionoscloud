# How to use a custom image

We are using k8s [image-builder](https://github.com/kubernetes-sigs/image-builder).

### Prerequisites

If you take a look at the [book](https://image-builder.sigs.k8s.io/capi/providers/raw), you can see following commands for building raw `qcow2` images:   
[Prerequisites](https://image-builder.sigs.k8s.io/capi/capi#prerequisites):
- Packer version >= 1.6.0
- Goss plugin for Packer version >= 1.2.0
- Ansible version >= 2.10.0
- Qemu plugin for Packer version >= 1.1.0


First, clone the repo `git clone git@github.com:kubernetes-sigs/image-builder.git`   
The build prerequisites for using image-builder for building raw images are managed by running:
```sh
cd image-builder/images/capi
make deps-qemu
```

### Build the image

Before we can build the image, we have to set the k8s version we want to build the image for.   
If you don't specify the version, the latest version, -2, will be used. If the latest is, e.g., 1.30, then 1.28 will be used.   

For example for k8s version `1.28.3`:

```sh
export PACKER_FLAGS="--var 'kubernetes_rpm_version=1.28.3' --var 'kubernetes_semver=v1.28.3' --var 'kubernetes_series=v1.28' --var 'kubernetes_deb_version=1.28.3-1.1'"
```

Now you can build the image:
```sh
make build-qemu-ubuntu-2204
```
In this case, for Ubuntu 22.04. Works also for `1804` or `2004`.   
The image creation takes quite some time, so be patient.   

The image will be created in the `output` directory.

Now your image has to be ported to qcow2 format:
```sh
qemu-img convert -O qcow2 <image> "<image>.qcow2"
```

### Upload image to IONOS Cloud

You can now upload the qcow2 image to IONOS Cloud via:   
**NOTE: You have to use basic auth (username:password). Token authentication is not working with FTP.**
```sh
ionosctl img upload -l txl -i <image>
```
`-l txl` is the location, txl in this case, available locations are: `fra, fkb, txl, lhr, las, ewr, vit`.


This process may take a considerable amount of time, and it's possible that you'll receive an email notifying you when your upload is scheduled.

Once the upload is completed successfully, you can locate the image either by using the `ionosctl img list` command or by navigating to DCD.

### Enabling cloud-init for your image

To enable cloud-init functionality for your image, you need to make some adjustments in DCD:

1. Go to Management -> Images & Snapshots -> select your image.
2. Set the "OS Type" to `Linux` and the "Cloud Init Support" to `V1 - User Data`, and save your changes.

You can also do that via Cloud API:
```sh
ionosctl img update --image-id <image-id> --cloudinit V1 --licence-type LINUX
```

### Enabling disk serial exposure

The disk serial number is required for dynamic volume provisioning plugins like a CSI driver to function properly.
User provided images do not have this enabled by default

Currently it's only possible to enable this using the REST API:

```sh
curl -H "Authorization: Bearer <JWT>" -H "Content-Type: application/json" -X PATCH \
    https://api.ionos.com/cloudapi/v6/images/<image-id> -d '{"exposeSerial": true}'
```

**NOTE**: All VMs that were created with the image before enabling the feature will need to be rebuilt in order for it to take effect.

Now, you can copy the ID of your image and set it as the `IONOSCLOUD_MACHINE_IMAGE_ID` environment variable. Your custom image will then be used.

> [!IMPORTANT]
> Please ensure to update the KUBERNETES_VERSION in your environment file (envfile) if it changes.
