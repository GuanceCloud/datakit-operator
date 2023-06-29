# Datakit-Operator

## Overview and Installation

Datakit Operator is a collaborative project between Datakit and Kubernetes orchestration. Its purpose is to assist in deploying Datakit more conveniently, as well as other functions such as verification and injection.

The details refer to [document](https://docs.guance.com/en/datakit/datakit-operator/).

Currently, Datakit-Operator provides the following functions:

- [x] Injection DDTrace Agent(Java/Python/JavaScript) and related environments.
- [x] Injection Sidecar logfwd to collect Pod logging.
- [x] Injection Profiler (Java/Python) and related environments.
- [x] Support task distribution for Datakit plugins.
   
Prerequisites:

- Recommended Kubernetes version 1.24.1 or above and internet access (to download yaml file and pull images).
- Ensure `MutatingAdmissionWebhook` and `ValidatingAdmissionWebhook` [controllers](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#prerequisites) are enabled.
- Ensure admissionregistration.k8s.io/v1 API is enabled.

Download [*datakit-operator.yaml*](https://static.guance.com/datakit-operator/datakit-operator.yaml), and follow these steps:

```
$ kubectl create namespace datakit
$ wget https://static.guance.com/datakit-operator/datakit-operator.yaml
$ kubectl apply -f datakit-operator.yaml
$ kubectl get pod -n datakit
NAME                               READY   STATUS    RESTARTS   AGE
datakit-operator-f948897fb-5w5nm   1/1     Running   0          15s
```

