## Overview

This project is based on the [Operator Framework][of-home], an open source toolkit to manage Kubernetes native applications, called Operators, in an effective, automated, and scalable way. Read more in the [introduction blog post][of-blog].

[Fortio][fortio-home] is a load testing tool.
Fortio runs at a specified query per second (qps) and records an histogram of execution time and calculates percentiles (e.g. p99 ie the response time such as 99% of the requests take less than that number (in seconds, SI unit)). It can run for a set duration, for a fixed number of calls, or until interrupted (at a constant target QPS, or max speed/load per connection/thread).

The name fortio comes from greek φορτίο which means load/burden.

## Installation

Run this command to deploy the operator
```sh
kubectl apply -f https://raw.githubusercontent.com/verfio/fortio-operator/master/deploy/fortio.yaml

customresourcedefinition.apiextensions.k8s.io "servers.fortio.verf.io" created
customresourcedefinition.apiextensions.k8s.io "loadtests.fortio.verf.io" created
customresourcedefinition.apiextensions.k8s.io "curltests.fortio.verf.io" created
serviceaccount "fortio-operator" created
clusterrolebinding.rbac.authorization.k8s.io "fortio-operator" created
role.rbac.authorization.k8s.io "fortio-operator" created
rolebinding.rbac.authorization.k8s.io "fortio-operator" created
deployment.apps "fortio-operator" created
configmap "fortio-data-dir" created
```
Verify that fortio-operator pod is up and running

```sh
kubectl get pods
NAME                              READY     STATUS    RESTARTS   AGE
fortio-operator-8fdc6d967-ssjk4   1/1       Running   0          33s
```

## LoadTest

Create LoadTest resource and define desired conditions, for example, [this YAML][fortio-loadtest] says that we want to test the https://verf.io for 10 seconds:

```yaml
apiVersion: fortio.verf.io/v1alpha1
kind: LoadTest
metadata:
  name: verfio
spec:
  duration: 10s
  url: "https://verf.io"
  action: load
```
Apply this file:

```sh
kubectl apply -f https://raw.githubusercontent.com/verfio/fortio-operator/master/deploy/crds/fortio_v1alpha1_loadtest_cr.yaml

loadtest.fortio.verf.io "verfio" created
```
Verify that Job to run the LoadTest was created and Pod successfully finished the required task:

```sh
kubectl get jobs
NAME         DESIRED   SUCCESSFUL   AGE
verfio-job   1         1            4m

kubectl get pods
NAME                              READY     STATUS      RESTARTS   AGE
fortio-operator-8fdc6d967-ssjk4   1/1       Running     0          15m
verfio-job-v8wl6                  0/1       Completed   0          5m
```

When test is finished, the result will be stored in the `fortio-data-dir` configmap:

```sh
kubectl get cm
NAME                   DATA      AGE
fortio-data-dir        1         19m
fortio-operator-lock   0         19m
```
Check the content of this data (output omitted):

```sh
kubectl describe cm fortio-data-dir
verfio_2018-12-22_155126.json:
----
{
  "RunType": "HTTP",
  "Labels": "verf.io , verfio-job-v8wl6",
  "StartTime": "2018-12-22T15:51:10.053834734Z",
  "RequestedQPS": "8",
  "RequestedDuration": "10s",
  "ActualQPS": 7.970731756747274,
  "ActualDuration": 10036719644,
  "NumThreads": 4,
  "Version": "1.3.1-pre",
  "DurationHistogram": {
    "Count": 80,
    "Min": 0.028049263,
    "Max": 0.073276722,
    "Sum": 2.9869050279999994,
    "Avg": 0.03733631284999999,
    "StdDev": 0.013932356831744559,
    "Data": [
      {
        "Start": 0.028049263,
        "End": 0.03,
        "Percent": 25,
        "Count": 20
      },
      {
        "Start": 0.03,
        "End": 0.035,
        "Percent": 72.5,
        "Count": 38
      },
      ...
```
In order to visualize the data run the Server.

## Server
Run this command to instruct fortio-operator to spin up the server:
```sh
kubectl apply -f https://raw.githubusercontent.com/verfio/fortio-operator/master/deploy/crds/fortio_v1alpha1_server_cr.yaml
server.fortio.verf.io "fortio-server" created
```
Check IP address of Server:
```sh
kubectl get service fortio-server
NAME            TYPE           CLUSTER-IP     EXTERNAL-IP      PORT(S)          AGE
fortio-server   LoadBalancer   10.27.255.49   IP_ADDRESS   8080:30269/TCP   1m
```
Navigate to specified address: http://{IP_ADDRESS}:8080/fortio/ to see the Fortio's UI and to http://{IP_ADDRESS}:8080/fortio/browse to see the list of saved results. Pick the existing one from the list and you will see the fancy diagram.

## Clean up

Delete Server
```sh
kubectl delete server fortio-server
server.fortio.verf.io "fortio-server" deleted
```

Delete LoadTest
```sh
kubectl delete loadtest verfio
loadtest.fortio.verf.io "verfio" deleted
```

Delete Operator:
```sh
kubectl delete -f https://raw.githubusercontent.com/verfio/fortio-operator/master/deploy/fortio.yaml
customresourcedefinition.apiextensions.k8s.io "servers.fortio.verf.io" deleted
customresourcedefinition.apiextensions.k8s.io "loadtests.fortio.verf.io" deleted
customresourcedefinition.apiextensions.k8s.io "curltests.fortio.verf.io" deleted
serviceaccount "fortio-operator" deleted
clusterrolebinding.rbac.authorization.k8s.io "fortio-operator" deleted
role.rbac.authorization.k8s.io "fortio-operator" deleted
rolebinding.rbac.authorization.k8s.io "fortio-operator" deleted
deployment.apps "fortio-operator" deleted
configmap "fortio-data-dir" deleted
```


[of-home]: https://github.com/operator-framework
[of-blog]: https://coreos.com/blog/introducing-operator-framework
[fortio-home]: https://github.com/fortio/fortio
[fortio-loadtest]: https://raw.githubusercontent.com/verfio/fortio-operator/master/deploy/crds/fortio_v1alpha1_loadtest_cr.yaml