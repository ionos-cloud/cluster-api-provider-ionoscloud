#/bin/bash

rm ./templates/clusterclass-template-replaced.yaml
sed 's/${CLUSTER_CLASS_NAME}/cluster-class/g' ./templates/clusterclass-template.yaml > ./templates/clusterclass-template-replaced.yaml
sed -i 's/${NAMESPACE}/default/g' ./templates/clusterclass-template-replaced.yaml

rm ./templates/cluster-template-topology-replaced.yaml
sed 's/${CLUSTER_CLASS_NAME}/cluster-class/g' ./templates/cluster-template-topology.yaml > ./templates/cluster-template-topology-replaced.yaml
sed -i 's/${NAMESPACE}/default/g' ./templates/cluster-template-topology-replaced.yaml

echo 'replace envs in ./templates/cluster-template-topology-replaced.yaml'
echo 'clusterctl alpha topology plan -f ./templates/clusterclass-template-replaced.yaml -f ./templates/cluster-template-topology-replaced.yaml -o output/'