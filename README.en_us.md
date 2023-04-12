# Datakit-Operator

## Overview and Installation

Datakit-Operator is a collaborative project between Datakit and Kubernetes orchestration. Its purpose is to assist in deploying Datakit more conveniently, as well as other functions such as verification and injection.

The details refer to [document](https://docs.guance.com/en/datakit/datakit-operator/).

Currently, Datakit-Operator provides the following functions:

- [x] Injection of `dd-lib` files and environment.
- [x] Injection of `logfwd` program and enabling log collection.
- [ ] Verify the configuration of Datakit.

Prerequisites:

- Recommended Kubernetes version is v1.24.1 or higher, and internet access is required (to download yaml files and images).
- Ensure that MutatingAdmissionWebhook and ValidatingAdmissionWebhook [admission controllers](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#prerequisites) are enabled.
- Ensure that the admissionregistration.k8s.io/v1 API is enabled.

```
$ kubectl create namespace datakit
$ wget https://static.guance.com/datakit-operator/datakit-operator.yaml
$ kubectl apply -f datakit-operator.yaml
$ kubectl get pod -n datakit
NAME                               READY   STATUS    RESTARTS   AGE
datakit-operator-f948897fb-5w5nm   1/1     Running   0          15s
```

