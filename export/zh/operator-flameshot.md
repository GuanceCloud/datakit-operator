# DataKit Operator 注入 Flameshot

DataKit Operator 从 [:octicons-tag-24: v1.8.0](operator-changelog.md#cl-1.8.0) 开始支持注入 Flameshot Sidecar。Flameshot 可以按计划或资源阈值采集 Java、Python 和 Go 应用的 Profiling 数据，用于替代旧版 Profiler 注入。

## 前置条件 {#flameshot-prerequisites}

- 集群已安装 [DataKit](https://docs.<<<custom_key.brand_main_domain>>>/datakit/datakit-daemonset-deploy/){:target="_blank"}。
- DataKit 已开启 [Profile 采集器](https://docs.<<<custom_key.brand_main_domain>>>/datakit/datakit-daemonset-deploy/#using-k8-env){:target="_blank"}。
- 目标 Pod 的安全策略允许 Sidecar 增加 `SYS_PTRACE` capability。
- 如果启用 Prometheus Annotation，DataKit 还需开启 KubernetesPrometheus，并启用 Pod Annotation 自动发现。

## Operator 配置 {#flameshot-usage}

在 `admission_inject_v2.flameshots` 中添加规则：

```json
{
    "admission_inject_v2": {
        "flameshots": [
            {
                "name": "flameshot-java",
                "namespace_selectors": ["^production$"],
                "label_selectors": ["profiling=flameshot"],
                "image": "{{.FlameshotImage}}",
                "envs": {
                    "FLAMESHOT_DATAKIT_ADDR": "http://datakit-service.datakit:9529/profiling/v1/input",
                    "FLAMESHOT_MONITOR_INTERVAL": "10s",
                    "FLAMESHOT_LOG_LEVEL": "info",
                    "FLAMESHOT_PROFILING_PATH": "/flameshot-data",
                    "FLAMESHOT_LOG_PATH": "/var/log/flameshot.log",
                    "FLAMESHOT_HTTP_LOCAL_IP": "{fieldRef:status.podIP}",
                    "FLAMESHOT_HTTP_LOCAL_PORT": "8089"
                },
                "processes": "[{\"service\":\"java-demo\",\"language\":\"java\",\"command\":\"^java\\\\b.*app\\\\.jar$\",\"events\":\"cpu\",\"duration\":\"30s\",\"cpu_usage_percent\":80}]",
                "enable_prometheus_annotations": true,
                "resources": {
                    "requests": {
                        "cpu": "100m",
                        "memory": "128Mi"
                    },
                    "limits": {
                        "cpu": "200m",
                        "memory": "256Mi"
                    }
                }
            }
        ]
    }
}
```

常用字段如下：

| 字段 | 说明 |
| --- | --- |
| `name` | 规则名称，用于日志定位，建议配置 |
| `namespace_selectors` | Namespace 正则数组 |
| `label_selectors` | Pod Label Selector 数组 |
| `image` | Flameshot Sidecar 镜像 |
| `envs` | Sidecar 环境变量 |
| `processes` | 必填且不能为空；进程匹配和采集策略的 JSON 数组字符串 |
| `enable_prometheus_annotations` | 是否自动添加 Flameshot 指标采集 Annotation，默认 `false` |
| `resources` | Sidecar 资源配置；缺失或非法时使用默认值 |

`FLAMESHOT_PROFILING_PATH` 和有效的 `FLAMESHOT_HTTP_LOCAL_PORT` 是注入必需项。缺少其中任意一项，或 `processes` 为空时，Operator 会跳过注入并记录 warning。

Selector 和 Annotation 的通用规则参见 [DataKit Operator 注入规则](datakit-operator.md#datakit-operator-inject)。`admission.datakit/flameshot.enabled: "false"` 可以为单个 Pod 禁用 Flameshot。

## 注入结果 {#flameshot-injection-result}

规则匹配后，Operator 会：

- 添加 `datakit-flameshot` Sidecar，并增加 `SYS_PTRACE` capability；
- 将 Pod 设置为共享进程命名空间，使 Sidecar 能发现业务进程；
- 创建 `flameshot-volume` EmptyDir，并挂载到所有普通容器的 `FLAMESHOT_PROFILING_PATH`；
- 将 `processes` 作为 `FLAMESHOT_PROCESSES` 注入 Sidecar；
- 将 Pod `restartPolicy` 设置为 `Always`。

Flameshot 会直接访问业务进程。上线前应确认 Pod Security Admission、容器安全策略以及应用所在环境允许上述变更。

## 采集配置 {#envs}

常用环境变量：

| 环境变量 | 说明 |
| --- | --- |
| `FLAMESHOT_DATAKIT_ADDR` | DataKit Profiling 接收地址 |
| `FLAMESHOT_MONITOR_INTERVAL` | 进程和资源监控间隔 |
| `FLAMESHOT_LOG_LEVEL` | Flameshot 日志级别 |
| `FLAMESHOT_PROFILING_PATH` | Profiling 临时文件共享目录，注入必需 |
| `FLAMESHOT_LOG_PATH` | Flameshot 日志路径 |
| `FLAMESHOT_HTTP_LOCAL_IP` | Flameshot HTTP 监听 IP |
| `FLAMESHOT_HTTP_LOCAL_PORT` | Flameshot HTTP 和指标端口，注入必需 |
| `FLAMESHOT_SERVICE` | 覆盖所有进程规则中的 service |
| `FLAMESHOT_TAGS` | 全局 Profiling 标签 |
| `FLAMESHOT_POD_CPU_LIMIT` | Pod CPU limit，单位为 millicore |
| `FLAMESHOT_POD_MEM_LIMIT` | Pod 内存 limit，单位为 MiB |

`processes` 支持 Java、Python 和 Go 的命令匹配、采集时长、CPU/内存阈值以及语言特定选项。Heap Dump 和对象存储上传也通过现有 `envs` 配置；敏感凭证建议使用 `{secretKeyRef:<SECRET>.<KEY>}`。完整字段参见 [Flameshot 文档](../integrations/flameshot.md)。

### Prometheus Annotation {#prom-anno}

当 `enable_prometheus_annotations: true` 时，Operator 会添加：

```yaml
prometheus.io/scrape: "true"
prometheus.io/port: "8089"
prometheus.io/scheme: "http"
prometheus.io/path: "/metrics"
prometheus.io/param_measurement: "flameshot"
```

端口取自 `FLAMESHOT_HTTP_LOCAL_PORT`。如果 Pod 已存在任意 `prometheus.io/` 开头的 Annotation，Operator 会保留用户配置，不添加以上 Annotation。

## Deployment 示例 {#flameshot-example}

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: java-demo
  namespace: production
spec:
  replicas: 1
  selector:
    matchLabels:
      app: java-demo
  template:
    metadata:
      labels:
        app: java-demo
        profiling: flameshot
      annotations:
        admission.datakit/flameshot.enabled: "true"
    spec:
      containers:
        - name: app
          image: example/java-demo:1.0.0
```

创建后检查：

```shell
kubectl -n production get pod -l app=java-demo -o jsonpath='{.items[0].spec.containers[*].name}'
kubectl -n production logs -l app=java-demo -c datakit-flameshot
```

结果应包含 `datakit-flameshot`。产生 Profiling 数据后，可在 <<<custom_key.brand_name>>> 的 Profiling 页面查看；没有数据时，先检查 Sidecar 日志、DataKit 地址、`processes` 的命令正则和目标进程权限。
