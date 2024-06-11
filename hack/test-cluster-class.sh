#/bin/bash

source templates/.env

clusterctl generate yaml --from templates/clusterclass-template.yaml > ./templates/clusterclass-template-replaced.yaml
clusterctl generate yaml --from templates/cluster-template-topology.yaml > ./templates/cluster-template-topology-replaced.yaml

echo "validate"
clusterctl alpha topology plan -f ./templates/clusterclass-template-replaced.yaml -f ./templates/cluster-template-topology-replaced.yaml -o output/

# echo "apply"
#kubectl apply -f ./templates/clusterclass-template-replaced.yaml
#kubectl apply -f ./templates/luster-template-topology-replaced.yaml