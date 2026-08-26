# DataKit Operator 注入 OpenTelemetry

DataKit Operator 从 [:octicons-tag-24: v1.9.0](operator-changelog.md#cl-1.9.0) 开始支持为 Java、Python 和 Node.js 应用注入 OpenTelemetry 自动探针。本功能使用 OpenTelemetry 官方自动注入镜像，并遵循其复制探针和设置启动环境的方式。

发布模板中的 `otels` 默认为空，OpenTelemetry 注入不会自动开启。启用时需要先添加本页所示规则。注入只发生在 Pod 创建时；Operator 会修改 Pod 中的所有普通业务容器，不修改业务 init Container，也不检测容器内的语言版本或 libc。

## 使用前准备 {#otel-prerequisites}

### 开启 DataKit OpenTelemetry 采集器 {#enable-datakit-otel}

DataKit 必须启用 `opentelemetry` input。例如，在 DataKit DaemonSet 的默认采集器列表中增加 `opentelemetry`：

```yaml
- name: ENV_DEFAULT_ENABLED_INPUTS
  value: statsd,dk,cpu,ddtrace,opentelemetry
```

OTLP Trace、Metric 和 Log 使用同一个 DataKit Service 和 `9529` 端口，但请求路径不同：

```text
Trace:  http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/traces
Metric: http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/metrics
Log:    http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/logs
```

本页的示例规则只开启 Trace。需要采集 Metric 或 Log 时，将对应的 `OTEL_METRICS_EXPORTER` 或 `OTEL_LOGS_EXPORTER` 从 `none` 改为 `otlp`，不需要配置新的 DataKit 地址。应用和自动探针本身还必须能够产生相应信号。

### 确认语言支持范围 {#otel-language-support}

| 语言 | 支持范围 | 默认镜像 |
| --- | --- | --- |
| Java | OpenTelemetry Java Agent 支持的 JVM | `{{.OTelJavaImage}}` |
| Python | Python 3.10 到 3.14；仅 glibc Linux | `{{.OTelPythonImage}}` |
| Node.js | Node.js 20.6 及以上；支持 glibc 和 Alpine/musl | `{{.OTelNodeJSImage}}` |

Python 2、Python 3.9 及以下和 Alpine/musl Python 镜像不在支持范围内。Node.js 当前支持常规 CommonJS 应用，不支持 ESM、bundler 或自定义 loader。OpenTelemetry 官方 Operator 目前尚未正式支持 PHP 自动注入。官方已提供 PHP 自动注入镜像，但尚未提供对应的 Instrumentation 配置和标准注入流程，因此 DataKit Operator 暂不支持 OTel PHP 注入。

Operator 不会自动判断这些条件。请使用互斥的 Namespace 或 Label Selector，只让符合条件的 Pod 匹配对应规则。

## Operator 配置 {#otel-config}

`otels` 与 `ddtraces` 平级，发布模板中的默认值为 `[]`。将下面的完整 Java 规则添加到该数组即可启用符合 Selector 条件的 Pod：

```json
{
    "admission_inject_v2": {
        "otels": [
            {
                "name": "otel-java",
                "language": "java",
                "namespace_selectors": ["^production$"],
                "label_selectors": ["admission.datakit/otel-language=java"],
                "check_annotation": false,
                "image": "{{.OTelJavaImage}}",
                "envs": {
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "OTEL_SERVICE_NAME": "{fieldRef:metadata.labels['app']}",
                    "OTEL_RESOURCE_ATTRIBUTES": "k8s.pod.name=$(POD_NAME),k8s.namespace.name=$(POD_NAMESPACE),k8s.node.name=$(NODE_NAME)",
                    "OTEL_TRACES_EXPORTER": "otlp",
                    "OTEL_LOGS_EXPORTER": "none",
                    "OTEL_METRICS_EXPORTER": "none",
                    "OTEL_EXPORTER_OTLP_PROTOCOL": "http/protobuf",
                    "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/traces",
                    "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/logs",
                    "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/metrics"
                },
                "resources": {
                    "requests": {
                        "cpu": "100m",
                        "memory": "64Mi"
                    },
                    "limits": {
                        "cpu": "500m",
                        "memory": "512Mi"
                    }
                }
            }
        ]
    }
}
```

环境变量保持配置顺序。上例中的 `POD_NAME`、`POD_NAMESPACE` 和 `NODE_NAME` 必须位于引用它们的 `OTEL_RESOURCE_ATTRIBUTES` 之前。

常用字段如下：

| 字段 | 说明 |
| --- | --- |
| `name` | 规则名称，用于日志定位，建议配置 |
| `language` | 必填；可选 `java`、`python` 或 `nodejs` |
| `namespace_selectors` | Namespace 正则数组 |
| `label_selectors` | Pod Label Selector 数组 |
| `check_annotation` | 是否要求 Pod 提供对应语言的版本注解，默认 `false` |
| `image` | 必填；OpenTelemetry 官方镜像或其私有仓库副本 |
| `envs` | 注入所有普通业务容器的环境变量 |
| `resources` | init Container 的资源配置；缺失或非法时使用默认值 |

Python 和 Node.js 规则的字段相同，只需修改 `name`、`language`、语言标签和镜像：

| 语言 | Label | 镜像 |
| --- | --- | --- |
| Python | `admission.datakit/otel-language=python` | `{{.OTelPythonImage}}` |
| Node.js | `admission.datakit/otel-language=nodejs` | `{{.OTelNodeJSImage}}` |

Selector 和 Annotation 的通用规则参见 [DataKit Operator 注入规则](datakit-operator.md#datakit-operator-inject)。同一个 Pod 匹配多条 OTel 规则时，Operator 只使用第一条符合 annotation 条件的规则；第一条规则的语言或镜像配置错误时也不会回退。

## 各语言的注入方式 {#otel-injection}

所有语言都会增加：

- `datakit-otel-lib-init` init Container；
- `datakit-otel-auto-instrument` EmptyDir 卷；
- 规则中配置的 `OTEL_*` 等环境变量。

不同语言的探针加载方式如下：

| 语言 | init Container 复制方式 | 挂载和启动环境 |
| --- | --- | --- |
| Java | 将镜像中的 `/javaagent.jar` 复制到共享卷 | 挂载 `/otel-auto-instrumentation-java`，向 `JAVA_TOOL_OPTIONS` 追加 `-javaagent:/otel-auto-instrumentation-java/javaagent.jar` |
| Python | 将镜像中的 `/autoinstrumentation/` 复制到共享卷 | 挂载 `/otel-auto-instrumentation-python`，在 `PYTHONPATH` 首尾加入自动初始化目录和探针目录，通过 `sitecustomize.py` 加载 |
| Node.js | 将镜像中的 `/autoinstrumentation/` 复制到共享卷 | 挂载 `/otel-auto-instrumentation-nodejs`，向 `NODE_OPTIONS` 追加 `--require /otel-auto-instrumentation-nodejs/autoinstrumentation.js` |

Operator 会保留已有的普通字符串值。如果对应启动环境使用 `valueFrom`、存在重复项、已经加载 Datadog 探针，或已有 OTel init Container、卷、挂载与预期冲突，Operator 会跳过整个 OTel 注入并记录 warning。Admission 仍保持 fail-open，不会阻止业务 Pod 创建。

## Annotation 与版本 {#otel-annotations}

`admission.datakit/otel.enabled: "false"` 可以为单个 Pod 禁用 OTel。该开关始终生效，与 `check_annotation` 无关。

当规则设置 `check_annotation: true` 时，Pod 还必须提供对应语言的版本注解：

| 语言 | 版本 Annotation |
| --- | --- |
| Java | `admission.datakit/otel-java-lib.version` |
| Python | `admission.datakit/otel-python-lib.version` |
| Node.js | `admission.datakit/otel-nodejs-lib.version` |

版本值只替换规则 `image` 的 tag，不会改变镜像仓库或名称。平台统一管理版本时建议保持 `check_annotation: false`。

## 与 DDTrace 的关系 {#otel-ddtrace-conflict}

同一个容器不应同时加载 DDTrace 和 OpenTelemetry 自动探针。DDTrace 与 OTel 规则同时匹配时，DDTrace 优先；DDTrace 已选中但注入失败时不会回退到 OTel。

建议为 OTel 工作负载显式关闭 DDTrace：

```yaml
admission.datakit/ddtrace.enabled: "false"
```

## `runAsNonRoot` 保护 {#otel-run-as-non-root}

OpenTelemetry 官方自动注入镜像的默认用户可能与业务 Pod 的安全策略不一致。Operator 会沿用第一个业务容器的 SecurityContext 创建 OTel init Container。

如果 init Container 的有效配置为 `runAsNonRoot: true`，但没有明确设置非零 `runAsUser`，Kubernetes 可能因无法确认镜像以非 root 用户运行而拒绝启动。为避免注入导致业务 Pod 卡在初始化阶段，Operator 会跳过整个 OTel 注入，并记录包含以下原因的 warning：

```text
reason=run_as_non_root_without_run_as_user
```

需要在这类 Pod 中启用 OTel 时，请为第一个业务容器或 Pod 明确配置一个与所用 OTel 镜像兼容的非零 `runAsUser`。在 `runAsNonRoot: true` 的情况下，私有镜像即使实际使用非 root 用户，缺少显式 `runAsUser` 时也会被保守跳过。

## Deployment 示例 {#otel-example}

下面的 Java Deployment 会匹配前面的配置：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: order-service
  namespace: production
spec:
  replicas: 1
  selector:
    matchLabels:
      app: order-service
  template:
    metadata:
      labels:
        app: order-service
        admission.datakit/otel-language: java
      annotations:
        admission.datakit/ddtrace.enabled: "false"
        admission.datakit/otel.enabled: "true"
    spec:
      containers:
        - name: app
          image: example/order-service:1.0.0
          ports:
            - name: http
              containerPort: 8080
```

将标签值改为 `python` 或 `nodejs`，即可匹配对应语言规则。业务镜像必须符合前述支持范围。

## 验证与排查 {#otel-verify}

创建 Pod 后检查注入结果：

```shell
kubectl -n production get pod -l app=order-service -o yaml
kubectl logs -n datakit deployment/datakit-operator
```

最终 Pod 应包含 `datakit-otel-lib-init`、`datakit-otel-auto-instrument`、对应语言的挂载、启动环境以及 `OTEL_*` 环境变量。随后向应用发起真实请求，并在页面查询：

```text
service:order-service
source:opentelemetry
```

常见问题：

- 已运行的 Pod 没有变化：重新创建 Pod；Operator 只处理 `CREATE`。
- 没有注入：检查 Namespace、Label、`check_annotation`、DDTrace 优先级和 Operator warning。
- init Container 拉取失败：官方镜像使用 `imagePullPolicy: Always`，检查 GHCR 网络，或将镜像同步到私有仓库后修改规则。
- 有注入但没有数据：确认 DataKit 已开启 `opentelemetry` input、OTLP 地址可达、业务框架受自动探针支持，并产生一次真实请求。
- 需要回滚：删除或禁用 OTel 规则，然后重新创建已经注入的 Pod。
