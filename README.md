# Datakit-Operator

*[English](./README.en_us.md) | 中文*

## 概述

Datakit Operator 是 Datakit 在 Kubernetes 编排的联动项目，旨在协助 Datakit 更方便的部署，以及其他诸如验证、注入的功能。

详情请参考[文档](https://docs.guance.com/datakit/datakit-operator/)。

目前 Datakit-Operator 提供以下功能：

- [x] 注入 DDTrace Agent（Java/Python/JavaScript）以及对应环境变量信息。
- [x] 注入 Sidecar logfwd 服务以采集容器内日志。
- [x] 注入 Profiler（Java/Python）以及对应环境变量信息。
- [x] 支持 Datakit 采集器的任务分发。

先决条件：

- 推荐 Kubernetes v1.24.1 及以上版本，且能够访问互联网（载 yaml 文件并拉取对应镜像）。
- 确保启用 `MutatingAdmissionWebhook` 和 `ValidatingAdmissionWebhook` [控制器](https://kubernetes.io/zh-cn/docs/reference/access-authn-authz/extensible-admission-controllers/#prerequisites)。
- 确保启用了 admissionregistration.k8s.io/v1 API。

下载 [*datakit-operator.yaml*](https://static.guance.com/datakit-operator/datakit-operator.yaml)，步骤如下：

```
$ kubectl create namespace datakit
$ wget https://static.guance.com/datakit-operator/datakit-operator.yaml
$ kubectl apply -f datakit-operator.yaml
$ kubectl get pod -n datakit
NAME                               READY   STATUS    RESTARTS   AGE
datakit-operator-f948897fb-5w5nm   1/1     Running   0          15s
```
