# DataKit Operator

---

:material-kubernetes:

---

DataKit Operator 通过 Kubernetes Admission Webhook 为新建 Pod 注入链路追踪、日志采集和性能分析组件，并提供集群内 Pod 查询和 DataKit 中心选举协调接口。

## 概述 {#overview}

DataKit Operator 提供以下功能：

| 功能 | 说明 |
| --- | --- |
| DDTrace 自动注入 | 支持 Java、Python、PHP 和 Node.js |
| OpenTelemetry 自动注入 | 从 v1.9.0 开始支持 Java、Python 和 Node.js |
| logfwd 注入 | 通过 Sidecar 采集未写入容器标准输出的文件日志 |
| Flameshot 注入 | 动态采集应用 Profiling 数据 |
| Profiler 注入 | 兼容旧版 async-profiler、py-spy 等注入方式 |
| Logging 配置注入 | 添加 `datakit/logs` 注解以及对应的文件卷挂载 |
| Cluster API | 代理查询集群内 Pod 数据，降低 DataKit 直接访问 API Server 的压力 |
| 中心选举协调 | 使用 Kubernetes Lease 为集群内 DataKit 裁决 Collection Leader |

注入发生在 Pod `CREATE` 阶段，不会修改已经运行的 Pod。修改 Operator 配置、注入镜像或工作负载注解后，需要重新创建 Pod 才能生效。

Admission 采用 fail-open：注入失败时 Operator 会记录日志，但不会阻止业务 Pod 创建。部署清单中的 webhook 同时使用 `failurePolicy: Ignore`。

## 先决条件 {#prerequisites}

- Kubernetes 需支持 `admissionregistration.k8s.io/v1`，推荐使用 Kubernetes v1.24 及以上版本。
- 集群需启用 `MutatingAdmissionWebhook` Admission Controller。
- 集群节点需能够拉取配置中的镜像；离线环境应提前将镜像同步到私有仓库。

## 安装 {#install}

<!-- markdownlint-disable MD046 -->
=== "Deployment"

    下载 [*datakit-operator.yaml*](https://static.<<<custom_key.brand_main_domain>>>/datakit-operator/datakit-operator.yaml){:target="_blank"} 并安装：

    ```shell
    kubectl create namespace datakit
    wget https://static.<<<custom_key.brand_main_domain>>>/datakit-operator/datakit-operator.yaml
    kubectl apply -f datakit-operator.yaml
    kubectl get pod -n datakit
    ```

    Pod 正常启动后应显示为 `Running`：

    ```text
    NAME                                READY   STATUS    RESTARTS   AGE
    datakit-operator-f948897fb-5w5nm    1/1     Running   0          15s
    ```

=== "Helm"

    Helm 版本需为 3.0 或以上。

    ```shell
    helm install datakit-operator datakit-operator \
        <<<% if custom_key.brand_key == 'guance' -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/datakit-operator \
        <<<% else -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/truewatch \
        <<<% endif -%>>>
        -n datakit --create-namespace
    ```

    查看部署状态：

    ```shell
    helm -n datakit list
    ```

    升级：

    ```shell
    helm -n datakit get values datakit-operator -a -o yaml > values.yaml
    helm upgrade datakit-operator datakit-operator \
        <<<% if custom_key.brand_key == 'guance' -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/datakit-operator \
        <<<% else -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/truewatch \
        <<<% endif -%>>>
        -n datakit \
        -f values.yaml
    ```

    卸载：

    ```shell
    helm uninstall datakit-operator -n datakit
    ```

???+ attention

    - Operator 程序与部署清单需要配套使用。升级 Operator 时，应同时更新 YAML 或 Helm Chart；仅启用中心选举时，也可按[增量升级说明](#central-election-upgrade)补充权限。
    - 出现 `InvalidImageName` 或镜像拉取失败时，请检查镜像地址、仓库权限和节点网络。
<!-- markdownlint-enable MD046 -->

### 配置说明 {#jsonconfig}

Operator 配置使用 JSON 格式。部署清单通常将配置保存在 ConfigMap 中，并通过 `ENV_JSON_CONFIG` 环境变量加载。

从 v1.8.0 开始使用 `admission_inject_v2`：

```json
{
    "server_listen": "0.0.0.0:9543",
    "log_level": "info",
    "admission_inject_v2": {
        "ddtraces": [],
        "otels": [],
        "logfwds": [],
        "flameshots": [],
        "profilers": []
    },
    "admission_mutate": {
        "loggings": []
    }
}
```

上例仅展示配置结构。实际发布模板默认保留一条 Java DDTrace 规则，匹配 `default` 命名空间中的 Pod；`otels` 默认为空，不会自动开启 OpenTelemetry 注入。这个默认值是兼容已有部署的运行策略，并不代表 Operator 只能处理 Java。

Operator 不会自动识别业务容器的语言。DDTrace 还支持 Python、PHP 和 Node.js，OpenTelemetry 支持 Java、Python 和 Node.js；启用这些能力时，应根据 [DDTrace 自动注入](operator-ddtrace.md)和 [OpenTelemetry 自动注入](operator-otel.md)文档添加使用互斥语言标签的规则。

旧版 `admission_inject` 配置仍然兼容。旧配置中有效的 `ddtrace`、`logfwd` 或 `profiler` 会分别覆盖对应的 v2 规则，升级时不要同时维护两套有效配置。

## Cluster API {#cluster-api}

DataKit Operator [:octicons-tag-24: v1.8.1](operator-changelog.md#cl-1.8.1) 及以后版本提供 Cluster API。该接口使用 Operator 的 Pod informer 缓存代理查询 Pod 数据，减少各 DataKit 实例直接访问 API Server 的压力。

Cluster API 默认开启。Operator 启动时会检查 ServiceAccount 的 Pod 读取权限；权限不足时相关路由不会注册，Pod 缓存同步完成前或同步失败时接口返回 `503`，但其他功能可以继续运行。所需最小权限如下：

```yaml
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
```

接口复用 Operator Service，默认地址为 `https://datakit-operator.datakit.svc:443`：

| 接口 | 说明 |
| --- | --- |
| `/v1/cluster/api/v1/pods` | 查询所有 Pod |
| `/v1/cluster/api/v1/namespaces/{namespace}/pods` | 查询指定命名空间中的 Pod |
| `/v1/cluster/api/v1/namespaces/{namespace}/pods/{name}` | 查询指定 Pod |

从 [:octicons-tag-24: v1.8.9](operator-changelog.md#cl-1.8.9) 开始，可以使用 `view=ebpf-v1` 获取面向 eBPF 的精简 Pod 结构：

```shell
curl -k "https://datakit-operator.datakit.svc:443/v1/cluster/api/v1/pods?view=ebpf-v1"
```

## DataKit 中心选举 {#central-election}

从 DataKit Operator v1.9.1 开始，可以替代 DataWay/Kodo 为 DataKit 提供中心选举。该功能只替代选举服务，采集数据仍通过 DataWay 上传。

该功能仅适用于 Kubernetes 环境，DataKit Operator 必须与 DataKit 部署在同一个 Kubernetes 集群中。

使用前需要注意：

1. **版本配套**：DataKit Operator 需要 v1.9.1 及以上版本，DataKit 需要 2.12.0 及以上版本，并手动配置环境变量。
1. **Lease 与 RBAC 权限**：DataKit Operator 使用 Kubernetes Lease 保存选举状态，需要相应的读写权限。Lease 是 Kubernetes 原生资源，无需安装 CRD，也无需手动创建 Lease 对象，DataKit Operator 会按需创建。

### 开启方式 {#central-election-config}

先升级 DataKit Operator 并补齐权限，再为同组选举的所有 DataKit 统一配置以下环境变量，然后重启 DataKit：

```yaml
- name: ENV_ENABLE_ELECTION
  value: "true"
- name: ENV_ELECTION_OPERATOR_URL
  value: "https://datakit-operator.datakit.svc:443"
```

示例使用默认的 Service 和 namespace；自定义部署请相应调整地址。

DataKit 仅在启动时判断是否使用 DataKit Operator。未配置地址、版本不支持、权限不足或暂时不可用时，继续使用原有 DataWay/Kodo 选举。运行期间不会切换；修改配置或恢复 DataKit Operator 后，需要重启 DataKit 才会重新判断。

切换选举方式时，先确认 DataKit Operator 的选举功能可用，再停止同组选举的 DataKit，完成统一配置后再启动。启动后应从日志确认它们均使用同一选举服务，避免普通滚动更新时两种选举服务混用而重复采集。

### 已有部署补充权限 {#central-election-upgrade}

新版默认 YAML 和 Helm Chart 已包含 Lease 的 Role/RoleBinding。已有部署无需替换整份 YAML：升级 DataKit Operator 后，可单独补充以下 RBAC。保存为 `datakit-operator-election-rbac.yaml`；自定义 namespace 或 ServiceAccount 时，请调整 `metadata.namespace`、`subjects.namespace` 和 `subjects.name`。Lease 会创建在 DataKit Operator 所在的 Kubernetes namespace 中。

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: datakit-operator-election
  namespace: datakit
rules:
- apiGroups: ["coordination.k8s.io"]
  resources: ["leases"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: datakit-operator-election
  namespace: datakit
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: datakit-operator-election
subjects:
- kind: ServiceAccount
  name: datakit-operator
  namespace: datakit
```

```shell
kubectl apply -f datakit-operator-election-rbac.yaml
kubectl -n datakit rollout restart deployment/datakit-operator
kubectl -n datakit rollout status deployment/datakit-operator
```

补充权限后，按上文配置并重启 DataKit。缺少 Lease 对象不会报错；缺少 Lease 权限会使中心选举不可用，但不影响 DataKit Operator 的其他服务。

## 注入规则 {#datakit-operator-inject}

每项注入配置都必须先通过 selector 匹配 Pod。Annotation 可以拒绝注入，或在启用 `check_annotation` 时进一步限定注入，但不能脱离 selector 单独触发注入。

### Selector 配置 {#selectors-injection}

`namespace_selectors` 和 `label_selectors` 都是数组：

- 同一数组中的多个 selector 按“或”匹配。
- 两个数组同时配置时，namespace 和 label 两个维度都必须匹配。
- 注入规则只配置一个维度时，仅使用该维度匹配；两个维度都未配置时禁用当前规则并继续检查后续规则。
- Logging mutation 规则需要同时配置两个维度，缺少任一维度时禁用当前规则。

`namespace_selectors` 使用 Go 正则表达式。字符串 `"*"` 是匹配全部命名空间的简写；精确匹配建议使用 `^` 和 `$`。`label_selectors` 使用 Kubernetes Label Selector 语法，并为 `=`、`==` 和 `!=` 扩展了 glob 匹配。

Selector 数组中的空字符串、纯空白、非法 namespace 正则以及非法或空约束 label selector 都是无效项。Operator 启动时会记录 warning 并忽略无效项；如果某个已配置维度没有剩余的有效项，则禁用当前规则并继续检查后续规则。需要显式匹配全部命名空间时请使用 `"*"`。

下面的规则只匹配 `production` 命名空间中带有 `admission.datakit/ddtrace-language=java` 标签的 Pod：

```json
{
    "namespace_selectors": ["^production$"],
    "label_selectors": ["admission.datakit/ddtrace-language=java"]
}
```

同一 Pod 可能匹配多条 DDTrace 或 OTel 规则。Operator 按配置顺序选择第一条符合 annotation 条件的规则，选中后不会因语言、镜像或其他配置错误回退到后续规则。多语言规则应使用互斥标签。

DDTrace 与 OTel 同时匹配时，DDTrace 优先；DDTrace 注入失败也不会回退到 OTel。

### Annotation 配置 {#annotation-injection}

Annotation 需要添加到 Pod，或 Deployment 等控制器的 `.spec.template.metadata.annotations` 中。

| Annotation | 作用 |
| --- | --- |
| `admission.datakit/enabled` | 控制 Operator 的全部 Pod 变更，优先级最高 |
| `admission.datakit/ddtrace.enabled` | 控制 DDTrace 注入 |
| `admission.datakit/otel.enabled` | 控制 OTel 注入 |
| `admission.datakit/logfwd.enabled` | 控制 logfwd 注入 |
| `admission.datakit/flameshot.enabled` | 控制 Flameshot 注入 |
| `admission.datakit/profiler.enabled` | 控制旧版 Profiler 注入 |

这些开关注解使用宽松语义：只有能够解析为 `false` 的值才会禁用相应功能；注解缺失或无法解析时按 `true` 处理。即使显式设置为 `true`，Pod 仍需匹配对应配置规则。

```yaml
metadata:
  annotations:
    admission.datakit/ddtrace.enabled: "false"
    admission.datakit/otel.enabled: "true"
```

### `check_annotation` {#check-annotation-config}

DDTrace、OTel、logfwd 和旧版 Profiler 规则支持 `check_annotation`：

| 值 | 行为 |
| --- | --- |
| `false` | 默认值。selector 匹配后不要求版本或旧版配置 annotation |
| `true` | selector 匹配后，还必须存在该规则对应的 annotation |

对应关系如下：

| 功能 | `check_annotation: true` 时要求的 Annotation |
| --- | --- |
| DDTrace | `admission.datakit/<language>-lib.version` |
| OTel | `admission.datakit/otel-<language>-lib.version` |
| Profiler | `admission.datakit/<language>-profiler.version` |
| logfwd | `admission.datakit/logfwd.instances` |

当 `check_annotation: true` 且 Pod 提供对应版本 annotation 时，DDTrace、OTel 和 Profiler 会替换规则中 `image` 的 tag，但不会改变镜像仓库和镜像名称。功能开关注解始终生效，不受 `check_annotation` 影响。

### 镜像拉取策略 {#image-pull-policy}

`admission_inject_v2` 下的 DDTrace、OTel、logfwd、Flameshot 和 Profiler 规则支持 `image_pull_policy`，与 `image` 同级。合法值为 `Always`、`IfNotPresent` 和 `Never`，区分大小写；缺失、空值或错误值（包括错误的 JSON 值类型）均使用 `Always`，错误值会记录 warning。

例如，将 `admission_inject_v2.ddtraces` 中对应的规则配置为：

```json
{
    "name": "ddtrace-java",
    "language": "java",
    "namespace_selectors": ["default"],
    "image": "{{.DDTraceJavaImage}}",
    "image_pull_policy": "IfNotPresent"
}
```

`IfNotPresent` 在节点已有镜像时复用本地镜像；`Never` 要求节点预先拥有镜像。建议使用固定版本，避免同名可变标签继续使用缓存中的旧镜像。该配置仅控制新注入的容器，Helm 的 `image.pullPolicy` 仍只控制 Operator 自身的镜像。

修改配置后重启 Operator，再重建业务 Pod。已有容器的策略不会被改写。废弃的 `admission_inject` 保持默认 `Always`；使用新配置时，需要移除会覆盖对应 v2 规则的有效旧配置。

## 支持的注入功能列表 {#supported-operator}

| 功能 | 文档 |
| --- | --- |
| DDTrace | [DDTrace 自动注入](operator-ddtrace.md) |
| OpenTelemetry | [OpenTelemetry 自动注入](operator-otel.md) |
| logfwd | [logfwd Sidecar 注入](operator-logfwd.md) |
| Flameshot | [Flameshot 注入](operator-flameshot.md) |
| async-profiler | [旧版 Java Profiler 注入](operator-asyncprofile.md) |
| py-spy | [旧版 Python Profiler 注入](operator-pyspy.md) |
| Logging | [日志采集配置注入](operator-logging.md) |

## 环境变量值引用 {#downwardapi}

注入规则的 `envs` 支持字面量，也支持将占位符转换为 Kubernetes 原生的 `fieldRef`、`resourceFieldRef` 和 `secretKeyRef`。

| 格式 | 说明 |
| --- | --- |
| `{fieldRef:metadata.name}` | Pod 名称 |
| `{fieldRef:metadata.namespace}` | Pod 命名空间 |
| `{fieldRef:metadata.uid}` | Pod UID |
| `{fieldRef:metadata.annotations['<KEY>']}` | 指定 Pod Annotation |
| `{fieldRef:metadata.labels['<KEY>']}` | 指定 Pod Label |
| `{fieldRef:spec.serviceAccountName}` | ServiceAccount 名称 |
| `{fieldRef:spec.nodeName}` | 节点名称 |
| `{fieldRef:status.hostIP}` | 节点主 IP |
| `{fieldRef:status.hostIPs}` | 节点双栈 IP |
| `{fieldRef:status.podIP}` | Pod 主 IP |
| `{resourceFieldRef:limits.cpu}` | 第一个业务容器的 CPU limit，单位为 CPU 核的 1/1000 |
| `{resourceFieldRef:limits.memory}` | 第一个业务容器的内存 limit，单位为 MiB |
| `{resourceFieldRef:requests.cpu}` | 第一个业务容器的 CPU request，单位为 CPU 核的 1/1000 |
| `{resourceFieldRef:requests.memory}` | 第一个业务容器的内存 request，单位为 MiB |
| `{secretKeyRef:<SECRET_NAME>.<KEY>}` | Pod 所在命名空间中的 Secret key |

环境变量保持配置顺序，后面的值可以通过 Kubernetes `$(VAR)` 语法引用前面定义的变量：

```json
{
    "envs": {
        "POD_NAME": "{fieldRef:metadata.name}",
        "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
        "RESOURCE_TAGS": "pod_name=$(POD_NAME),pod_namespace=$(POD_NAMESPACE)"
    }
}
```

无法识别的占位符会作为普通字符串注入。`resourceFieldRef` 只引用第一个业务容器；如果该容器没有声明对应的 request 或 limit，该环境变量不会注入。

### `{secretKeyRef:*}` {#secretkeyref}

Secret 引用格式如下：

```text
{secretKeyRef:<secret-name>.<key>}
```

例如：

```json
{
    "envs": {
        "ACCESS_KEY_ID": "{secretKeyRef:flameshot-oss.access_key_id}",
        "ACCESS_KEY_SECRET": "{secretKeyRef:flameshot-oss.access_key_secret}"
    }
}
```

Operator 不会读取 Secret，Kubernetes 会在容器启动时解析引用。Secret 必须和业务 Pod 位于同一个命名空间。Secret 或 key 不存在时，Pod 会进入 `CreateContainerConfigError`。由于 `.` 用作 Secret 名称与 key 的分隔符，此格式中的 Secret 名称不能包含 `.`。

## FAQ {#faq}

### 如何禁用特定 Pod 的全部变更？ {#disable-inject}

在 Pod 模板中添加：

```yaml
admission.datakit/enabled: "false"
```

### 注入为什么没有生效？ {#debug}

按以下顺序检查：

1. Pod 是否为配置更新后重新创建的新 Pod。
1. Namespace 和 Label 是否匹配同一条规则。
1. `admission.datakit/enabled` 及功能开关注解是否为 `false`。
1. `check_annotation: true` 时，是否存在正确的版本或配置 annotation。
1. Operator 日志中是否有 selector、镜像、环境变量或安全上下文冲突的 warning。

### 在 AWS EKS 环境需要注意什么？ {#aws-eks}

EKS 控制平面需要能够访问 Operator webhook 的 `9543` 端口。注入不生效时，请检查集群安全组和网络策略是否允许该方向的流量。
