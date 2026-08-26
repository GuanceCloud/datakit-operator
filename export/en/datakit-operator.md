# DataKit Operator

---

:material-kubernetes:

---

DataKit Operator uses a Kubernetes Admission Webhook to inject tracing, log collection, and profiling components into newly created Pods. It also provides an in-cluster Pod query API.

## Overview {#overview}

DataKit Operator provides the following features:

| Feature | Description |
| --- | --- |
| DDTrace automatic injection | Supports Java, Python, PHP, and Node.js |
| OpenTelemetry automatic injection | Supports Java, Python, and Node.js starting with v1.9.0 |
| logfwd injection | Collects file logs not written to container standard output through a Sidecar |
| Flameshot injection | Dynamically collects application Profiling data |
| Profiler injection | Supports legacy injection methods such as async-profiler and py-spy |
| Logging configuration injection | Adds the `datakit/logs` Annotation and corresponding file volume mounts |
| Cluster API | Proxies queries for Pod data in the cluster, reducing direct access to the API Server from DataKit |

Injection occurs during the Pod `CREATE` phase and does not modify running Pods. After changing the Operator configuration, an injection image, or workload Annotations, recreate the Pod for the change to take effect.

Admission is fail-open: if injection fails, the Operator logs the error but does not block creation of the application Pod. The webhook in the deployment manifest also uses `failurePolicy: Ignore`.

## Prerequisites {#prerequisites}

- Kubernetes must support `admissionregistration.k8s.io/v1`; Kubernetes v1.24 or later is recommended.
- The `MutatingAdmissionWebhook` Admission Controller must be enabled in the cluster.
- Cluster nodes must be able to pull the configured images. In offline environments, synchronize the images to a private registry in advance.

## Installation {#install}

<!-- markdownlint-disable MD046 -->
=== "Deployment"

    Download [*datakit-operator.yaml*](https://static.<<<custom_key.brand_main_domain>>>/datakit-operator/datakit-operator.yaml){:target="_blank"} and install it:

    ```shell
    kubectl create namespace datakit
    wget https://static.<<<custom_key.brand_main_domain>>>/datakit-operator/datakit-operator.yaml
    kubectl apply -f datakit-operator.yaml
    kubectl get pod -n datakit
    ```

    After the Pod starts successfully, its status should be `Running`:

    ```text
    NAME                                READY   STATUS    RESTARTS   AGE
    datakit-operator-f948897fb-5w5nm    1/1     Running   0          15s
    ```

=== "Helm"

    Helm 3.0 or later is required.

    ```shell
    helm install datakit-operator datakit-operator \
        <<<% if custom_key.brand_key == 'guance' -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/datakit-operator \
        <<<% else -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/truewatch \
        <<<% endif -%>>>
        -n datakit --create-namespace
    ```

    Check the deployment status:

    ```shell
    helm -n datakit list
    ```

    Upgrade:

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

    Uninstall:

    ```shell
    helm uninstall datakit-operator -n datakit
    ```

???+ attention

    - The Operator binary and deployment manifest must be used together. When upgrading the Operator, update the YAML or Helm Chart at the same time.
    - If `InvalidImageName` occurs or an image cannot be pulled, check the image address, registry permissions, and node network.
<!-- markdownlint-enable MD046 -->

### Configuration {#jsonconfig}

The Operator uses JSON configuration. Deployment manifests generally store the configuration in a ConfigMap and load it through the `ENV_JSON_CONFIG` environment variable.

`admission_inject_v2` is used starting with v1.8.0:

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

The preceding example shows only the configuration structure. The distributed deployment templates retain one Java DDTrace rule that matches Pods in the `default` Namespace; `otels` is empty, so OpenTelemetry injection is not enabled automatically. This default is a runtime policy that preserves compatibility with existing deployments and does not mean that the Operator supports only Java.

The Operator does not detect the language in application containers automatically. DDTrace also supports Python, PHP, and Node.js, while OpenTelemetry supports Java, Python, and Node.js. To enable these capabilities, follow the [DDTrace automatic injection](operator-ddtrace.md) and [OpenTelemetry automatic injection](operator-otel.md) documentation and add rules that use mutually exclusive language labels.

The legacy `admission_inject` configuration remains supported. A valid `ddtrace`, `logfwd`, or `profiler` entry in the legacy configuration overrides the corresponding v2 rules. Do not maintain two valid configurations for the same feature during an upgrade.

## Cluster API {#cluster-api}

DataKit Operator [:octicons-tag-24: v1.8.1](operator-changelog.md#cl-1.8.1) and later provide the Cluster API. This API uses the Operator's Pod informer cache to proxy Pod data queries, reducing direct access to the API Server from each DataKit instance.

The Cluster API is enabled by default. At startup, the Operator checks whether its ServiceAccount has permission to read Pods. If permission is insufficient or cache synchronization fails, the related routes are not registered, but other features can continue to run. The minimum required permissions are:

```yaml
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
```

The API uses the Operator Service. Its default address is `https://datakit-operator.datakit.svc:443`:

| Endpoint | Description |
| --- | --- |
| `/v1/cluster/api/v1/pods` | Queries all Pods |
| `/v1/cluster/api/v1/namespaces/{namespace}/pods` | Queries Pods in a specified Namespace |
| `/v1/cluster/api/v1/namespaces/{namespace}/pods/{name}` | Queries a specified Pod |

Starting with [:octicons-tag-24: v1.8.9](operator-changelog.md#cl-1.8.9), use `view=ebpf-v1` to obtain a reduced Pod representation for eBPF:

```shell
curl -k "https://datakit-operator.datakit.svc:443/v1/cluster/api/v1/pods?view=ebpf-v1"
```

## Injection Rules {#datakit-operator-inject}

Every injection configuration must first match a Pod through selectors. An Annotation can reject injection or, when `check_annotation` is enabled, further restrict injection. It cannot trigger injection independently of selectors.

### Selector Configuration {#selectors-injection}

`namespace_selectors` and `label_selectors` are both arrays:

- Multiple selectors in the same array are matched with OR.
- When both arrays are configured, both the Namespace and Label dimensions must match.
- When an injection rule configures only one dimension, only that dimension is used. If neither dimension is configured, the current rule is disabled and evaluation continues with subsequent rules.
- Logging mutation rules require both dimensions. If either dimension is missing, the current rule is disabled.

`namespace_selectors` uses Go regular expressions. The string `"*"` is shorthand for matching all Namespaces; use `^` and `$` for exact matches. `label_selectors` uses Kubernetes Label Selector syntax and extends `=`, `==`, and `!=` with glob matching.

Empty strings, whitespace-only values, invalid Namespace regular expressions, and invalid or empty-constraint Label selectors are invalid entries. At startup, the Operator logs a warning and ignores each invalid entry. If a configured dimension has no valid entries left, the current rule is disabled and evaluation continues with subsequent rules. Use `"*"` to explicitly match all Namespaces.

The following rule matches only Pods in the `production` Namespace that have the `admission.datakit/ddtrace-language=java` label:

```json
{
    "namespace_selectors": ["^production$"],
    "label_selectors": ["admission.datakit/ddtrace-language=java"]
}
```

The same Pod may match multiple DDTrace or OTel rules. The Operator selects the first rule in configuration order that satisfies the Annotation conditions. Once selected, it does not fall back to subsequent rules because of an invalid language, image, or other configuration error. Use mutually exclusive labels for rules targeting different languages.

When DDTrace and OTel both match, DDTrace takes precedence. If DDTrace injection fails, the Operator does not fall back to OTel.

### Annotation Configuration {#annotation-injection}

Add Annotations to the Pod or to `.spec.template.metadata.annotations` in a controller such as a Deployment.

| Annotation | Purpose |
| --- | --- |
| `admission.datakit/enabled` | Controls all Operator changes to the Pod; has the highest priority |
| `admission.datakit/ddtrace.enabled` | Controls DDTrace injection |
| `admission.datakit/otel.enabled` | Controls OTel injection |
| `admission.datakit/logfwd.enabled` | Controls logfwd injection |
| `admission.datakit/flameshot.enabled` | Controls Flameshot injection |
| `admission.datakit/profiler.enabled` | Controls legacy Profiler injection |

These switch Annotations use permissive semantics: only a value that can be parsed as `false` disables the corresponding feature. A missing or unparseable Annotation is treated as `true`. Even when explicitly set to `true`, the Pod must still match the corresponding configuration rule.

```yaml
metadata:
  annotations:
    admission.datakit/ddtrace.enabled: "false"
    admission.datakit/otel.enabled: "true"
```

### `check_annotation` {#check-annotation-config}

DDTrace, OTel, logfwd, and legacy Profiler rules support `check_annotation`:

| Value | Behavior |
| --- | --- |
| `false` | Default. No version or legacy configuration Annotation is required after the selectors match |
| `true` | After the selectors match, the Annotation corresponding to the rule must also be present |

The mappings are:

| Feature | Annotation required when `check_annotation: true` |
| --- | --- |
| DDTrace | `admission.datakit/<language>-lib.version` |
| OTel | `admission.datakit/otel-<language>-lib.version` |
| Profiler | `admission.datakit/<language>-profiler.version` |
| logfwd | `admission.datakit/logfwd.instances` |

When `check_annotation: true` and the Pod provides the corresponding version Annotation, DDTrace, OTel, and Profiler replace the tag in the rule's `image` without changing the image registry or name. Feature switch Annotations always apply, regardless of `check_annotation`.

## Supported Injection Features {#supported-operator}

| Feature | Documentation |
| --- | --- |
| DDTrace | [DDTrace automatic injection](operator-ddtrace.md) |
| OpenTelemetry | [OpenTelemetry automatic injection](operator-otel.md) |
| logfwd | [logfwd Sidecar injection](operator-logfwd.md) |
| Flameshot | [Flameshot injection](operator-flameshot.md) |
| async-profiler | [Legacy Java Profiler injection](operator-asyncprofile.md) |
| py-spy | [Legacy Python Profiler injection](operator-pyspy.md) |
| Logging | [Log collection configuration injection](operator-logging.md) |

## Environment Variable Value References {#downwardapi}

The `envs` field in an injection rule supports literal values and placeholders that are converted to native Kubernetes `fieldRef`, `resourceFieldRef`, and `secretKeyRef` references.

| Format | Description |
| --- | --- |
| `{fieldRef:metadata.name}` | Pod name |
| `{fieldRef:metadata.namespace}` | Pod Namespace |
| `{fieldRef:metadata.uid}` | Pod UID |
| `{fieldRef:metadata.annotations['<KEY>']}` | Specified Pod Annotation |
| `{fieldRef:metadata.labels['<KEY>']}` | Specified Pod Label |
| `{fieldRef:spec.serviceAccountName}` | ServiceAccount name |
| `{fieldRef:spec.nodeName}` | Node name |
| `{fieldRef:status.hostIP}` | Primary node IP |
| `{fieldRef:status.hostIPs}` | Dual-stack node IPs |
| `{fieldRef:status.podIP}` | Primary Pod IP |
| `{resourceFieldRef:limits.cpu}` | CPU limit of the first application container, in millicores |
| `{resourceFieldRef:limits.memory}` | Memory limit of the first application container, in MiB |
| `{resourceFieldRef:requests.cpu}` | CPU request of the first application container, in millicores |
| `{resourceFieldRef:requests.memory}` | Memory request of the first application container, in MiB |
| `{secretKeyRef:<SECRET_NAME>.<KEY>}` | Secret key in the Pod's Namespace |

Environment variables preserve configuration order. Later values can reference earlier variables using Kubernetes `$(VAR)` syntax:

```json
{
    "envs": {
        "POD_NAME": "{fieldRef:metadata.name}",
        "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
        "RESOURCE_TAGS": "pod_name=$(POD_NAME),pod_namespace=$(POD_NAMESPACE)"
    }
}
```

Unrecognized placeholders are injected as ordinary strings. `resourceFieldRef` references only the first application container. If that container does not declare the corresponding request or limit, the environment variable is not injected.

### `{secretKeyRef:*}` {#secretkeyref}

The Secret reference format is:

```text
{secretKeyRef:<secret-name>.<key>}
```

For example:

```json
{
    "envs": {
        "ACCESS_KEY_ID": "{secretKeyRef:flameshot-oss.access_key_id}",
        "ACCESS_KEY_SECRET": "{secretKeyRef:flameshot-oss.access_key_secret}"
    }
}
```

The Operator does not read the Secret; Kubernetes resolves the reference when the container starts. The Secret must be in the same Namespace as the application Pod. If the Secret or key does not exist, the Pod enters `CreateContainerConfigError`. Because `.` separates the Secret name from the key, the Secret name in this format cannot contain `.`.

## FAQ {#faq}

### Disable Pod Changes {#disable-inject}

Add the following to the Pod template:

```yaml
admission.datakit/enabled: "false"
```

### Why Did Injection Not Take Effect? {#debug}

Check the following in order:

1. Whether the Pod was recreated after the configuration update.
1. Whether its Namespace and Label match the same rule.
1. Whether `admission.datakit/enabled` or a feature switch Annotation is `false`.
1. When `check_annotation: true`, whether the correct version or configuration Annotation is present.
1. Whether the Operator logs contain warnings about selectors, images, environment variables, or security context conflicts.

### What Should I Consider on AWS EKS? {#aws-eks}

The EKS control plane must be able to access port `9543` on the Operator webhook. If injection does not take effect, check whether the cluster security groups and network policies allow traffic in that direction.
