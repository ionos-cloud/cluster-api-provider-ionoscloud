#!/bin/bash

# This script is used to delete resources that were created during a GitHub workflow that was cancelled.
# GitHub doesn't give enough time for the test to gracefully terminate after a workflow cancellation. This
# results in leftover resources.

set -o pipefail

if [[ "${CI}" != "true" ]]; then
  echo "error: This script only makes sense when running from a CI environment. Make sure that if it is, the CI environment variable is set."
  exit 1
fi

if [[ -z "${IONOS_TOKEN}" ]]; then
  echo "error: The IONOS_TOKEN environment variable must be set."
  exit 1
fi

if [[ -n "${DATACENTER_ID}" ]]; then
    bin/ionosctl dc delete --datacenter-id "${DATACENTER_ID}"
fi

if [[ -n "${IP_BLOCK_ID}" ]]; then
    bin/ionosctl ipb delete --ipblock-id "${IP_BLOCK_ID}"
fi
