#/bin/bash

ROOT_DIR=$(cd $(realpath $(dirname $0))/.. && pwd)

if [[ ! -d "./output" ]]; then
    echo "create output directory"
    mkdir "./output"
fi

if [[ ! -f "./.envfile" ]]; then
    echo "create .envfile from envfile.example"
    exit 1
fi
source ./.envfile

clusterctl generate yaml --from templates/clusterclass-template.yaml > ./output/generated-clusterclass-template-replaced.yaml
clusterctl generate cluster $CLUSTER_NAME --from templates/cluster-template-topology.yaml > ./output/generated-cluster-template-topology.yaml
#
#echo "validate"

echo "done"
#clusterctl alpha topology plan -f ./templates/clusterclass-template-replaced.yaml -f ./templates/cluster-template-topology-replaced.yaml -o output/

# echo "apply"
#kubectl apply -f ./templates/clusterclass-template-replaced.yaml
#kubectl apply -f ./templates/cluster-template-topology-replaced.yaml
