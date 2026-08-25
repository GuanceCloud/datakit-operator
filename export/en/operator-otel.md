# Injecting OpenTelemetry with DataKit Operator

Starting with [:octicons-tag-24: v1.9.0](operator-changelog.md#cl-1.9.0), DataKit Operator supports injecting OpenTelemetry automatic instrumentation into Java, Python, and Node.js applications. This feature uses the official OpenTelemetry automatic instrumentation images and follows their approach for copying instrumentation and configuring the startup environment.

Injection occurs only when a Pod is created. The Operator modifies all regular application containers in the Pod. It does not modify application init Containers or detect language versions or libc inside containers.

## Before You Begin {#otel-prerequisites}

### Enable the DataKit OpenTelemetry Input {#enable-datakit-otel}

The `opentelemetry` input must be enabled in DataKit. For example, add `opentelemetry` to the default enabled input list in the DataKit DaemonSet:

```yaml
- name: ENV_DEFAULT_ENABLED_INPUTS
  value: statsd,dk,cpu,ddtrace,opentelemetry
```

OTLP Traces, Metrics, and Logs use the same DataKit Service and port `9529`, but different request paths:

```text
Trace:  http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/traces
Metric: http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/metrics
Log:    http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/logs
```

The default Operator template enables only Traces. To collect Metrics or Logs, change the corresponding `OTEL_METRICS_EXPORTER` or `OTEL_LOGS_EXPORTER` from `none` to `otlp`; no new DataKit address is required. The application and automatic instrumentation must also be able to produce the corresponding signals.

### Confirm Language Support {#otel-language-support}

| Language | Supported range | Default image |
| --- | --- | --- |
| Java | JVM supported by the OpenTelemetry Java Agent | `{{.OTelJavaImage}}` |
| Python | Python 3.10 to 3.14; glibc Linux only | `{{.OTelPythonImage}}` |
| Node.js | Node.js 20.6 and later; supports glibc and Alpine/musl | `{{.OTelNodeJSImage}}` |

Python 2, Python 3.9 and earlier, and Alpine/musl Python images are unsupported. Node.js currently supports conventional CommonJS applications, but not ESM, bundlers, or custom loaders. The official OpenTelemetry Operator does not yet formally support PHP automatic injection. Although an official PHP automatic-instrumentation image is available, the Operator does not yet provide the corresponding Instrumentation configuration or a standard injection flow, so DataKit Operator does not currently support OTel injection for PHP.

The Operator does not detect these conditions automatically. Use mutually exclusive Namespace or Label Selectors so that only compatible Pods match each rule.

## Operator Configuration {#otel-config}

`otels` is at the same level as `ddtraces`. The following is a complete Java rule:

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

Environment variables preserve configuration order. In the preceding example, `POD_NAME`, `POD_NAMESPACE`, and `NODE_NAME` must appear before `OTEL_RESOURCE_ATTRIBUTES`, which references them.

Common fields:

| Field | Description |
| --- | --- |
| `name` | Rule name, recommended for locating relevant logs |
| `language` | Required; one of `java`, `python`, or `nodejs` |
| `namespace_selectors` | Array of Namespace regular expressions |
| `label_selectors` | Array of Pod Label Selectors |
| `check_annotation` | Whether the Pod must provide a version Annotation for the corresponding language; defaults to `false` |
| `image` | Required; official OpenTelemetry image or a copy in a private registry |
| `envs` | Environment variables injected into all regular application containers |
| `resources` | Resource configuration for the init Container; defaults are used when missing or invalid |

Python and Node.js rules use the same fields. Change only `name`, `language`, the language label, and the image:

| Language | Label | Image |
| --- | --- | --- |
| Python | `admission.datakit/otel-language=python` | `{{.OTelPythonImage}}` |
| Node.js | `admission.datakit/otel-language=nodejs` | `{{.OTelNodeJSImage}}` |

For the general rules governing selectors and Annotations, see [DataKit Operator injection rules](datakit-operator.md#datakit-operator-inject). If a Pod matches multiple OTel rules, the Operator uses only the first rule that satisfies the Annotation conditions. It does not fall back if that rule has an invalid language or image configuration.

## Injection by Language {#otel-injection}

All languages add:

- The `datakit-otel-lib-init` init Container.
- The `datakit-otel-auto-instrument` EmptyDir volume.
- The `OTEL_*` and other environment variables configured in the rule.

The instrumentation loading method differs by language:

| Language | init Container copy method | Mount and startup environment |
| --- | --- | --- |
| Java | Copies `/javaagent.jar` from the image to the shared volume | Mounts `/otel-auto-instrumentation-java` and appends to `JAVA_TOOL_OPTIONS` the value `-javaagent:/otel-auto-instrumentation-java/javaagent.jar` |
| Python | Copies `/autoinstrumentation/` from the image to the shared volume | Mounts `/otel-auto-instrumentation-python`, adds the automatic initialization and instrumentation directories to the beginning and end of `PYTHONPATH`, and loads through `sitecustomize.py` |
| Node.js | Copies `/autoinstrumentation/` from the image to the shared volume | Mounts `/otel-auto-instrumentation-nodejs` and appends to `NODE_OPTIONS` the value `--require /otel-auto-instrumentation-nodejs/autoinstrumentation.js` |

The Operator preserves existing ordinary string values. If the corresponding startup environment uses `valueFrom`, contains duplicates, already loads Datadog instrumentation, or an existing OTel init Container, volume, or mount conflicts with the expected value, the Operator skips the entire OTel injection and logs a warning. Admission remains fail-open and does not block creation of the application Pod.

## Annotations and Versions {#otel-annotations}

`admission.datakit/otel.enabled: "false"` disables OTel for an individual Pod. This switch always applies, regardless of `check_annotation`.

When a rule sets `check_annotation: true`, the Pod must also provide the version Annotation for the corresponding language:

| Language | Version Annotation |
| --- | --- |
| Java | `admission.datakit/otel-java-lib.version` |
| Python | `admission.datakit/otel-python-lib.version` |
| Node.js | `admission.datakit/otel-nodejs-lib.version` |

The version value replaces only the tag in the rule's `image`; it does not change the image registry or name. When the platform manages versions centrally, keeping `check_annotation: false` is recommended.

## Relationship with DDTrace {#otel-ddtrace-conflict}

The same container should not load DDTrace and OpenTelemetry automatic instrumentation simultaneously. When DDTrace and OTel rules both match, DDTrace takes precedence. If DDTrace is selected but injection fails, the Operator does not fall back to OTel.

Explicitly disable DDTrace for OTel workloads:

```yaml
admission.datakit/ddtrace.enabled: "false"
```

## `runAsNonRoot` Safeguard {#otel-run-as-non-root}

The default user in official OpenTelemetry automatic instrumentation images may be incompatible with the application Pod's security policy. The Operator applies the SecurityContext of the first application container to the OTel init Container.

If the effective configuration of the init Container is `runAsNonRoot: true` but does not explicitly set a nonzero `runAsUser`, Kubernetes may refuse to start it because it cannot confirm that the image runs as a non-root user. To prevent injection from leaving the application Pod stuck during initialization, the Operator skips the entire OTel injection and logs a warning containing the following reason:

```text
reason=run_as_non_root_without_run_as_user
```

To enable OTel in such a Pod, explicitly configure a nonzero `runAsUser` compatible with the OTel image on the first application container or the Pod. When `runAsNonRoot: true`, the Operator conservatively skips injection without an explicit `runAsUser`, even if a private image actually runs as a non-root user.

## Deployment Example {#otel-example}

The following Java Deployment matches the preceding configuration:

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

Change the label value to `python` or `nodejs` to match the corresponding language rule. The application image must meet the support requirements described above.

## Verification and Troubleshooting {#otel-verify}

After creating the Pod, inspect the injection result:

```shell
kubectl -n production get pod -l app=order-service -o yaml
kubectl logs -n datakit deployment/datakit-operator
```

The resulting Pod should contain `datakit-otel-lib-init`, `datakit-otel-auto-instrument`, the mounts and startup environment for the corresponding language, and the `OTEL_*` environment variables. Then send an actual request to the application and query the page with:

```text
service:order-service
source:opentelemetry
```

Common issues:

- A running Pod did not change: recreate it; the Operator processes only `CREATE`.
- Nothing was injected: check the Namespace, Label, `check_annotation`, DDTrace precedence, and Operator warnings.
- The init Container cannot pull its image: official images use `imagePullPolicy: Always`. Check GHCR connectivity, or synchronize the image to a private registry and update the rule.
- Injection succeeded but no data appears: confirm that the `opentelemetry` input is enabled in DataKit, the OTLP address is reachable, the application framework is supported by automatic instrumentation, and an actual request has been sent.
- Rollback is required: remove or disable the OTel rule, then recreate Pods that were already injected.
