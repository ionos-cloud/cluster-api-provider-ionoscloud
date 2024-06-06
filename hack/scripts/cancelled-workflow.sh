#!/bin/bash

# This script is used to delete resources that were created during a GitHub workflow that was cancelled.
# GitHub doesn't give enough time for the test to gracefully terminate after a workflow cancellation. This
# results in leftover resources.

set -o errexit
set -o pipefail

if [[ "${CI}" != "true" ]]; then
  echo "error: This script only makes sense when running from a CI environment. Make sure that if it is, the CI environment variable is set."
  exit 1
fi

if [[ -z "${IONOS_TOKEN}" ]]; then
  echo "error: The IONOS_TOKEN environment variable must be set."
  exit 1
fi

IONOS_API_URL="https://api.ionos.com/cloudapi/v6/"
ERROR=0

# Function to make a DELETE request and handle the response
function delete_resource {
    URL="${IONOS_API_URL}/$1/$2" # api_url/{resource_type}/{resource_id}
    RESPONSE=$(curl -s -I -w "%{http_code}" -XDELETE -H "Authorization: Bearer ${IONOS_TOKEN}" "${URL}")
    RESPONSE_CODE="$(echo "${RESPONSE}" | tail -n 1)"
    if [[ "${RESPONSE_CODE}" == "202" ]]; then
        LOCATION="$(echo "${RESPONSE}" | grep location | awk '{print $2}')"
        echo "Deletion for $1/$2 requested. Check the status of the request at ${LOCATION}"
    elif [[ "${RESPONSE_CODE}" == "404" ]]; then
        echo "Resource $1/$2 does not exist."
    else
        echo "error: Received unexpected status code ${RESPONSE_CODE} for DELETE request to ${URL}"
        ERROR=1
    fi
}

if [[ -n "${DATACENTER_ID}" ]]; then
    delete_resource datacenters "${DATACENTER_ID}"
fi

if [[ -n "${IP_BLOCK_ID}" ]]; then
    delete_resource ipblocks "${IP_BLOCK_ID}"
fi

if [[ "${ERROR}" == "1" ]]; then
    echo "error: One or more resources could not be deleted."
    exit 1
fi
