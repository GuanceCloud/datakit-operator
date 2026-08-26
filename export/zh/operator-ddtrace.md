# DataKit Operator 注入 DDTrace

DataKit Operator 在 Pod 创建时注入 DDTrace 自动探针，支持 Java、Python、PHP 和 Node.js。Operator 会为 Pod 添加 `datakit-lib-init` init Container 和共享卷 `/datadog-lib`，并修改所有普通业务容器的启动环境。已有 Pod 不会被修改，需要重新创建后才能生效。

## 使用说明 {#datakit-operator-inject-lib-usage}

使用前，请先[安装 DataKit Operator](datakit-operator.md#install)，并确认业务容器可以访问 DataKit 的 Trace 接收地址。

发布模板默认保留一条 Java DDTrace 规则，匹配 `default` 命名空间中的所有 Pod。这是为了兼容已有部署，并不代表只支持 Java。Operator 不会自动识别业务容器的语言；配置 Python、PHP 或 Node.js 时，需要同时把默认 Java 规则改成使用互斥语言标签，否则它会先匹配这些 Pod，使后续语言规则无法生效。

### 支持范围与镜像 {#ddtrace-lib-image-selection}

| 语言 | 业务运行时 | 默认镜像 |
| --- | --- | --- |
| Java | 支持 DDTrace Java Agent 的 JVM | `{{.DDTraceJavaImage}}` |
| Python | Python 3.7 | `{{.DDTracePython37Image}}` |
| Python | Python 3.8 | `{{.DDTracePython38Image}}` |
| Python | Python 3.9 到 3.14 | `{{.DDTracePythonImage}}` |
| PHP | Linux GNU 或 musl | `{{.DDTracePHPImage}}` |
| Node.js | Node.js 16 | `{{.DDTraceNodeJS16Image}}` |
| Node.js | Node.js 18 到 25 | `{{.DDTraceNodeJSImage}}` |

镜像版本必须与业务容器中的语言运行时兼容。离线环境可以将镜像同步到私有仓库，并在规则的 `image` 中填写完整地址。

PHP 规则还需要配置 `php_loader_flavor`：glibc 镜像使用 `linux-gnu`，Alpine 等 musl 镜像使用 `linux-musl`。Operator 不会自动检测业务镜像使用的 libc；未配置或配置错误时会回退到 `linux-gnu`。

PHP init Container 会复制对应 libc 的 loader 配置。业务进程启动后，Datadog loader 再根据 PHP 版本、ABI 以及 ZTS/NTS 模式加载兼容的 `.so` 文件；Operator 本身只选择 libc 类型，不检查 PHP 运行时。

### 配置规则 {#ddtrace-config}

下面的配置同时包含 Java、Python、PHP 和 Node.js 规则。编辑现有 `jsonconfig` 时，将其中的 `ddtraces` 数组整体替换到 `admission_inject_v2.ddtraces`，保留 `admission_inject_v2` 下的其他配置。不要把这些规则直接追加到发布模板的默认 Java 规则之后，否则默认规则会优先匹配。

所有规则使用 `"namespace_selectors": ["*"]`，但只有带对应语言标签的 Pod 才会匹配，因此可以跨 Namespace 使用而不会自动注入未标记的 Pod。示例中的 Python 镜像适用于 Python 3.9 到 3.14，PHP 使用 `linux-gnu`，Node.js 镜像适用于 Node.js 18 到 25；其他运行时需要根据前表替换镜像或 `php_loader_flavor`。

```json
{
    "admission_inject_v2": {
        "ddtraces": [
            {
                "name": "ddtrace-java",
                "language": "java",
                "namespace_selectors": ["*"],
                "label_selectors": ["admission.datakit/ddtrace-language=java"],
                "check_annotation": false,
                "image": "{{.DDTraceJavaImage}}",
                "envs": {
                    "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DD_TRACE_AGENT_PORT": "9529",
                    "DD_JMXFETCH_STATSD_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DD_JMXFETCH_STATSD_PORT": "8125",
                    "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
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
            },
            {
                "name": "ddtrace-python",
                "language": "python",
                "namespace_selectors": ["*"],
                "label_selectors": ["admission.datakit/ddtrace-language=python"],
                "check_annotation": false,
                "image": "{{.DDTracePythonImage}}",
                "envs": {
                    "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DD_TRACE_AGENT_PORT": "9529",
                    "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
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
            },
            {
                "name": "ddtrace-php",
                "language": "php",
                "php_loader_flavor": "linux-gnu",
                "namespace_selectors": ["*"],
                "label_selectors": ["admission.datakit/ddtrace-language=php"],
                "check_annotation": false,
                "image": "{{.DDTracePHPImage}}",
                "envs": {
                    "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DD_TRACE_AGENT_PORT": "9529",
                    "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
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
            },
            {
                "name": "ddtrace-nodejs",
                "language": "nodejs",
                "namespace_selectors": ["*"],
                "label_selectors": ["admission.datakit/ddtrace-language=nodejs"],
                "check_annotation": false,
                "image": "{{.DDTraceNodeJSImage}}",
                "envs": {
                    "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DD_TRACE_AGENT_PORT": "9529",
                    "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
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

为 Pod 设置且只设置一个对应语言标签，例如：

```yaml
metadata:
  labels:
    app: payment-service
    admission.datakit/ddtrace-language: python
```

常用字段如下：

| 字段 | 说明 |
| --- | --- |
| `name` | 规则名称，用于日志定位，建议配置 |
| `language` | 必填；可选 `java`、`python`、`php` 或 `nodejs` |
| `namespace_selectors` | Namespace 正则数组 |
| `label_selectors` | Pod Label Selector 数组 |
| `check_annotation` | 是否要求 Pod 提供对应语言的版本注解，默认 `false` |
| `image` | 必填；语言库 init Container 镜像 |
| `envs` | 注入所有普通业务容器的环境变量 |
| `resources` | init Container 的资源配置；缺失或非法时使用默认值 |
| `php_loader_flavor` | 仅 PHP 使用，可选 `linux-gnu` 或 `linux-musl` |

Selector、Annotation、默认资源以及环境变量引用的通用规则，参见 [DataKit Operator 注入规则](datakit-operator.md#datakit-operator-inject)。多语言规则应使用互斥标签，避免一个 Pod 同时匹配多条规则。

## 注入方式 {#ddtrace-injection}

| 语言 | Operator 对业务容器的修改 |
| --- | --- |
| Java | 挂载 `/datadog-lib`，向 `JAVA_TOOL_OPTIONS` 追加 `-javaagent:/datadog-lib/dd-java-agent.jar` |
| Python | 挂载 `/datadog-lib`，将 `/datadog-lib/` 添加到 `PYTHONPATH` 开头 |
| PHP | 挂载 `/datadog-lib`，配置 `DD_LOADER_PACKAGE_PATH` 和 `PHP_INI_SCAN_DIR`，加载 `dd_library_loader.ini` |
| Node.js | 挂载 `/datadog-lib`，向 `NODE_OPTIONS` 追加 `--require=/datadog-lib/node_modules/dd-trace/init` |

Operator 会保留业务容器中已有的同名启动参数。若 `JAVA_TOOL_OPTIONS`、`PYTHONPATH`、`PHP_INI_SCAN_DIR` 或 `NODE_OPTIONS` 使用 Kubernetes `valueFrom`，Operator 无法安全合并字符串，会跳过整个 DDTrace 注入并记录 warning，避免只留下无效的 init Container。

规则中的普通环境变量不会覆盖业务容器已有的同名变量。`DD_TAGS` 是例外：当两边都是普通字符串时，Operator 会合并标签。

## Annotation 与版本 {#check-annotation-config}

`admission.datakit/ddtrace.enabled: "false"` 可以为单个 Pod 禁用 DDTrace。该开关始终生效，与 `check_annotation` 无关。

当规则设置 `check_annotation: true` 时，Pod 还必须提供对应语言的版本注解：

| 语言 | 版本 Annotation |
| --- | --- |
| Java | `admission.datakit/java-lib.version` |
| Python | `admission.datakit/python-lib.version` |
| PHP | `admission.datakit/php-lib.version` |
| Node.js | `admission.datakit/nodejs-lib.version` |

版本值只会替换规则 `image` 的 tag，不会改变镜像仓库或名称。因此它只能用于切换同一镜像的版本。

如果多条 DDTrace 规则同时匹配，Operator 按配置顺序使用第一条符合 annotation 条件的规则。DDTrace 与 OpenTelemetry 同时匹配时，DDTrace 优先；DDTrace 已选中但注入失败时不会回退到 OpenTelemetry。不要为同一 Pod 同时启用两种自动探针。

## Deployment 示例 {#anno-demo}

下面的 Deployment 会匹配前面的 Java 规则：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: java-demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: java-demo
  template:
    metadata:
      labels:
        app: java-demo
        admission.datakit/ddtrace-language: java
      annotations:
        admission.datakit/ddtrace.enabled: "true"
    spec:
      containers:
        - name: app
          image: example/java-demo:1.0.0
```

如需明确禁用 DDTrace，只需将注解改为：

```yaml
admission.datakit/ddtrace.enabled: "false"
```

## 验证与排查 {#ddtrace-verify}

重新创建 Pod 后，检查最终 Pod：

```shell
kubectl get pod <pod-name> -o jsonpath='{.spec.initContainers[*].name}'
kubectl get pod <pod-name> -o yaml
kubectl logs -n datakit deployment/datakit-operator
```

应看到 `datakit-lib-init`、`datakit-auto-instrument` 卷、对应语言的挂载和启动环境。随后还需要发起一次真实请求，并在 DataKit monitor 或 <<<custom_key.brand_name>>> 页面确认 Trace 数据。

常见问题：

- 已运行的 Pod 没有变化：重新创建 Pod；Operator 只处理 `CREATE`。
- 没有注入：检查 Namespace、Label 和 `check_annotation` 是否同时满足。
- init Container 拉取失败：检查规则中的镜像地址、凭证和集群网络。
- init Container 成功但没有 Trace：检查业务进程是否保留注入的启动环境，以及 `DD_AGENT_HOST`、`DD_TRACE_AGENT_PORT` 是否可达。
- PHP 启动失败：确认业务镜像使用 glibc 还是 musl，并设置正确的 `php_loader_flavor`。
