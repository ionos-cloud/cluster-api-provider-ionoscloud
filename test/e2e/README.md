# End-to-End tests

This directory contains end-to-end tests for the provider. The tests are adapted from the
[Cluster API end-to-end tests][cluster-api-test-repo].

## Running the tests

To run the tests, you need to have the following tools installed:

- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/)

Besides the tools, the following environment variables need to be set:

- `IONOS_TOKEN`: The IONOS API token
- `CONTROL_PLANE_ENDPOINT_LOCATION`: The location for the control plane endpoint and machine image.
- `IONOSCLOUD_MACHINE_IMAGE_ID`: The ID of the machine image to be used for the nodes. MUST be in the location specified by `CONTROL_PLANE_ENDPOINT_LOCATION`.

Before running the tests, you need to build the provider image.

```shell
make docker-build
```

To run the tests, execute the following command:

```shell
make test-e2e
```

To skip the conformance tests, set `GINKGO_LABEL` to `!Conformance`.

```shell
make test-e2e GINKGO_LABEL='!Conformance'
```

If you want to skip the cleanup of the resources after the tests, you can set the `make ``SKIP_CLEANUP` variable to `true`.

```shell
make test-e2e SKIP_CLEANUP=true
```

## Developing tests

Besides the guidelines in the [Developing E2E tests](https://cluster-api.sigs.k8s.io/developer/e2e), it is recommended
to check the [Cluster API end-to-end tests][cluster-api-test-repo] as what you might want to write might be already
available there. If so, feel free to copy the test and adapt it to the provider.

[cluster-api-test-repo]: https://github.com/kubernetes-sigs/cluster-api/tree/main/test
