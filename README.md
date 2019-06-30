# Apexords-Operator in Kubernetes
This is operator to automate Apex [Oracle Application Express](https://apex.oracle.com) 19.1 and Ords [oracle rest data service](https://www.oracle.com/tools/technologies/faq-rest-data-services.html) via kubernetes [CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) ,it create a brand new Oracle 19c database statefulset,apex, ords  deployment plus load balancer in the kubernetes cluster

## How to test on local machine
* check kubectl cluster-info  is working properly 
* git clone https://github.com/HenryXie1/apexords-operator
* make install
* make run    ( it will run controller locally while communicating with K8S master)
* kubectl apply -f config/sample/theapexords_v1_apexords.yaml

## How to test in remote kubernetes cluster
* check kubectl cluster-info  is working properly 
* git clone https://github.com/HenryXie1/apexords-operator
* make install
* make docker-build docker-push IMG=<some-registry>/apexords-controller  
* Modify image locations on yaml files under config/default/
* kubectl apply -f config/sample/theapexords_v1_apexords.yaml
