# DataKit Operator 注入日志采集配置

DataKit Operator 可以为新建 Pod 添加 `datakit/logs` Annotation，并根据文件日志路径创建或复用 EmptyDir 卷。这样无需在每个 Deployment 中重复维护日志 Annotation 和目录挂载。

该功能适用于 DataKit 直接采集 Kubernetes Pod 文件日志。需要 Sidecar 将日志转发到 DataKit 时，请使用 [logfwd 注入](operator-logfwd.md)。

## Operator 配置 {#logging-config}

在 `admission_mutate.loggings` 中添加规则：

```json
{
    "admission_mutate": {
        "loggings": [
            {
                "namespace_selectors": ["^middleware$"],
                "label_selectors": ["app=logging"],
                "config": "[{\"disable\":false,\"type\":\"file\",\"path\":\"/var/log/app/*.log\",\"source\":\"logging-demo\"}]"
            }
        ]
    }
}
```

| 字段 | 说明 |
| --- | --- |
| `namespace_selectors` | Namespace 正则数组，必须至少配置一个可匹配项 |
| `label_selectors` | Pod Label Selector 数组，必须至少配置一个可匹配项 |
| `config` | 写入 `datakit/logs` 的 JSON 数组字符串 |

Namespace 和 Label 两个维度必须同时匹配。同一维度中的多个 selector 按“或”匹配；多条规则按配置顺序使用第一条匹配项。

`config` 必须是有效 JSON。Operator 会解析其中未禁用的 `type: "file"` 配置，从 `path` 提取目录并进行挂载。`stdout` 配置只写入 Annotation，不需要新增卷。

## 注入结果 {#logging-injection-result}

对于匹配上例的 Pod，Operator 会添加：

```yaml
metadata:
  annotations:
    datakit/logs: '[{"disable":false,"type":"file","path":"/var/log/app/*.log","source":"logging-demo"}]'
spec:
  containers:
    - name: app
      volumeMounts:
        - name: datakit-logs-volume-0
          mountPath: /var/log/app
  volumes:
    - name: datakit-logs-volume-0
      emptyDir: {}
```

Operator 会将目录挂载到所有普通业务容器：

- 如果该路径已经挂载 EmptyDir，则复用原有卷。
- 如果没有对应挂载，则创建新的 EmptyDir。
- 如果该路径已经使用其他类型的卷，则记录 warning 并跳过该路径。
- 如果 Pod 已有 `datakit/logs` Annotation，则保留用户配置，不再修改 Annotation 或卷。

## Deployment 示例 {#logging-example}

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: logging-demo
  namespace: middleware
spec:
  replicas: 1
  selector:
    matchLabels:
      app: logging
  template:
    metadata:
      labels:
        app: logging
    spec:
      containers:
        - name: app
          image: nginx:1.25
```

创建 Pod 后检查：

```shell
kubectl -n middleware get pod -l app=logging -o yaml
kubectl logs -n datakit deployment/datakit-operator
```

最终 Pod 应包含 `datakit/logs` Annotation，以及 `/var/log/app` 对应的 EmptyDir 和挂载。修改规则后需要重新创建 Pod 才能生效。
