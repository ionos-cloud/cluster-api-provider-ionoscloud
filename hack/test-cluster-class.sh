#/bin/bash

ROOT_DIR=$(cd $(realpath $(dirname $0))/.. && pwd)

if [[ ! -d "${ROOT_DIR}/output" ]]; then
    echo "create output directory"
    mkdir "${ROOT_DIR}/output"
fi

if [[ ! -f "${ROOT_DIR}/.envfile" ]]; then
    echo "create .envfile from envfile.example"
    exit 1
fi
source ${ROOT_DIR}/.envfile

echo "Setup Cluster API with ClusterResourceSet and ClusterTopology"
echo "export EXP_CLUSTER_RESOURCE_SET=\"true\""
echo "export CLUSTER_TOPOLOGY=\"true\""
echo "clusterclt init --infrastructure=ionos-cloud"

GENERATED_CLUSTER_CLASS_FILE="${ROOT_DIR}/output/generated-clusterclass-template.yaml"
GENERATED_CLUSTER_FILE="${ROOT_DIR}/output/generated-cluster-template-topology-calico.yaml"

clusterctl generate yaml --from "${ROOT_DIR}/templates/clusterclass-template.yaml" > $GENERATED_CLUSTER_CLASS_FILE
clusterctl generate cluster $CLUSTER_NAME -n $NAMESPACE --from "${ROOT_DIR}/templates/cluster-template-topology-calico.yaml" > $GENERATED_CLUSTER_FILE

echo "Validate generated resources"
clusterctl alpha topology plan -f $GENERATED_CLUSTER_CLASS_FILE -f $GENERATED_CLUSTER_FILE -o "${ROOT_DIR}/output/"

echo "Apply cluster class resources"
echo "kubectl apply -f $GENERATED_CLUSTER_CLASS_FILE"
echo "Apply cluster which uses the cluster class resources"
echo "kubectl apply -f $GENERATED_CLUSTER_FILE"

echo "Installing calico cni"
echo "make crs-calico"
echo "kubectl apply -f ${ROOT_DIR}/templates/crs/cni/calico.yaml"

echo "Done"
