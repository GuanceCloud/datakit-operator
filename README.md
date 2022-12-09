# Datakit-Operator

## 概述

Datakit-Operator 是 Datakit 在 Kubernetes 编排的联动项目，旨在协助 Datakit 更方便的部署，以及其他诸如验证、注入的功能。

目前 Datakit-Operator 提供以下功能：

- [ ] 针对特殊 Pod，提供注入 `dd-lib` 文件和 environment 的功能
- [x] 负责创建和更新 Datakit 即相关 Pod 的编排
- [x] 验证 Datakit 的配置

## 安装

需要 Kubernetes v1.24.1 及以上版本，且能够访问互联网（下载 yaml 文件和 Image）。

```
$ wget http://zhuyun-static-files-production.oss-cn-hangzhou-internal.aliyuncs.com/datakit-operator/datakit-operator.yaml
$ kubectl apply -f datakit-operator.yaml
```
