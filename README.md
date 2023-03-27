# Datakit-Operator

*[English](./README.en_us.md) | 中文*

## 概述

Datakit-Operator 是 Datakit 在 Kubernetes 编排的联动项目，旨在协助 Datakit 更方便的部署，以及其他诸如验证、注入的功能。

目前 Datakit-Operator 提供以下功能：

- [x] 针对特殊 Pod，提供注入 `dd-lib` 文件和 environment 的功能，参见[文档](./docs/admission-mutate.md)
- [ ] 负责创建和更新 Datakit 即相关 Pod 的编排
- [ ] 验证 Datakit 的配置

先决条件：

- 推荐 Kubernetes v1.24.1 及以上版本，且能够访问互联网（下载 yaml 文件和 pull Image）。
- 确保启用 MutatingAdmissionWebhook 和 ValidatingAdmissionWebhook [控制器](https://kubernetes.io/zh-cn/docs/reference/access-authn-authz/extensible-admission-controllers/#prerequisites)。
- 确保启用了 admissionregistration.k8s.io/v1 API。

```
$ kubectl create namespace datakit
$ wget https://static.guance.com/datakit-operator/datakit-operator.yaml
$ kubectl apply -f datakit-operator.yaml
$ kubectl get pod -n datakit
NAME                               READY   STATUS    RESTARTS   AGE
datakit-operator-f948897fb-5w5nm   1/1     Running   0          15s
```
