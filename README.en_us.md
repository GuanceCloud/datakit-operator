# Datakit-Operator

## Overview and Installation

Datakit-Operator is a collaborative project between Datakit and Kubernetes orchestration. Its purpose is to assist in deploying Datakit more conveniently, as well as other functions such as verification and injection.

Currently, Datakit-Operator provides the following functions:

- [x] Provide the ability to inject `dd-lib` files and environments for special Pods, refer to [document](./docs/admission-mutate.en_us.md).
- [ ] Responsible for creating and updating Datakit-related Pod scheduling
- [ ] Verify the configuration of Datakit

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

