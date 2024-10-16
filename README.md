# Kafka-Keda-Golang


This repo contains demo code for a Kafka Producer and a Consumer application.

Both apps only print out Unix Timestamps as messages

These apps have liveness/readiness probes + Prometheus Metrics.

## Demo Prereq:
```bash
#Add repos as needed
helm repo add strimzi https://strimzi.io/charts/
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add kedacore https://kedacore.github.io/charts
helm repo add marthydavid https://marthydavid.github.io/helm-charts
helm repo update
```
```bash
# Install Prometheus Operator
helm upgrade --install kube-prometheus-stack -n monitoring --create-namespace prometheus-community/kube-prometheus-stack

# Install strimzi
helm upgrade --install --version 0.43.0 strimzi-kafka-operator -n strimzi-system --create-namespace strimzi/strimzi-kafka-operator

# Install Keda
helm upgrade --install --version 2.15.1 keda -n keda --create-namespace kedacore/keda

# Install producer
helm upgrade --install producer -n demo --create-namespace marthydavid/producer --set replicaCount=0 --set producer.kafka.topic=demo

# Install consumer
helm upgrade --install consumer -n demo --create-namespace marthydavid/consumer --set replicaCount=0 --set producer.kafka.topic=demo
```

Every application should be installed a short summary:

```bash
kubectl get deployment,sts,pods -n monitoring
kubectl get deployment,sts,pods -n strimzi-system
kubectl get deployment,sts,pods -n keda
kubectl get deployment,sts,pods -n demo
```

# Demo

```bash
kubectl apply -f example

```
This will deploy the needed Kafka, KafkaTopics, KafkaMetrics, Prometheus PodMonitors, Grafana Dashboard, Keda Scaler

Now producer should be scaled up:
```bash
kubectl scale deployment -n demo producer --replicas=1
```
