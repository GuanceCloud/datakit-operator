# Injecting Flameshot with DataKit Operator

Starting with [:octicons-tag-24: v1.8.0](operator-changelog.md#cl-1.8.0), DataKit Operator supports injecting the Flameshot Sidecar. Flameshot can collect Profiling data from Java, Python, and Go applications on a schedule or when resource thresholds are reached. It replaces the legacy Profiler injection.

## Prerequisites {#flameshot-prerequisites}

- [DataKit](https://docs.<<<custom_key.brand_main_domain>>>/datakit/datakit-daemonset-deploy/){:target="_blank"} is installed in the cluster.
- The [Profile collector](https://docs.<<<custom_key.brand_main_domain>>>/datakit/datakit-daemonset-deploy/#using-k8-env){:target="_blank"} is enabled in DataKit.
- The target Pod's security policy allows the Sidecar to add the `SYS_PTRACE` capability.
- If Prometheus Annotations are enabled, KubernetesPrometheus must also be enabled in DataKit, with automatic discovery through Pod Annotations enabled.

## Operator Configuration {#flameshot-usage}

Add a rule to `admission_inject_v2.flameshots`:

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

Common fields:

| Field | Description |
| --- | --- |
| `name` | Rule name, recommended for locating relevant logs |
| `namespace_selectors` | Array of Namespace regular expressions |
| `label_selectors` | Array of Pod Label Selectors |
| `image` | Flameshot Sidecar image |
| `envs` | Sidecar environment variables |
| `processes` | Required and cannot be empty; JSON array string containing process matching and collection policies |
| `enable_prometheus_annotations` | Whether to add Flameshot metric collection Annotations automatically; defaults to `false` |
| `resources` | Sidecar resource configuration; defaults are used when missing or invalid |

`FLAMESHOT_PROFILING_PATH` and a valid `FLAMESHOT_HTTP_LOCAL_PORT` are required for injection. If either is missing, or if `processes` is empty, the Operator skips injection and logs a warning.

For the general rules governing selectors and Annotations, see [DataKit Operator injection rules](datakit-operator.md#datakit-operator-inject). `admission.datakit/flameshot.enabled: "false"` disables Flameshot for an individual Pod.

## Injection Result {#flameshot-injection-result}

After a rule matches, the Operator:

- Adds the `datakit-flameshot` Sidecar with the `SYS_PTRACE` capability.
- Enables a shared process namespace for the Pod so the Sidecar can discover application processes.
- Creates the `flameshot-volume` EmptyDir and mounts it at `FLAMESHOT_PROFILING_PATH` in all regular containers.
- Injects `processes` into the Sidecar as `FLAMESHOT_PROCESSES`.
- Sets the Pod `restartPolicy` to `Always`.

Flameshot accesses application processes directly. Before deployment, confirm that Pod Security Admission, container security policies, and the application environment allow these changes.

## Collection Configuration {#envs}

Common environment variables:

| Environment variable | Description |
| --- | --- |
| `FLAMESHOT_DATAKIT_ADDR` | DataKit Profiling receiver address |
| `FLAMESHOT_MONITOR_INTERVAL` | Process and resource monitoring interval |
| `FLAMESHOT_LOG_LEVEL` | Flameshot log level |
| `FLAMESHOT_PROFILING_PATH` | Shared directory for temporary Profiling files; required for injection |
| `FLAMESHOT_LOG_PATH` | Flameshot log path |
| `FLAMESHOT_HTTP_LOCAL_IP` | Flameshot HTTP listen IP |
| `FLAMESHOT_HTTP_LOCAL_PORT` | Flameshot HTTP and metrics port; required for injection |
| `FLAMESHOT_SERVICE` | Overrides service in all process rules |
| `FLAMESHOT_TAGS` | Global Profiling tags |
| `FLAMESHOT_POD_CPU_LIMIT` | Pod CPU limit, in thousandths of a CPU core |
| `FLAMESHOT_POD_MEM_LIMIT` | Pod memory limit, in MiB |

`processes` supports command matching, collection duration, CPU and memory thresholds, and language-specific options for Java, Python, and Go. Heap Dump and object storage uploads are also configured through the existing `envs`; use `{secretKeyRef:<SECRET>.<KEY>}` for sensitive credentials. For all fields, see the [Flameshot documentation](../integrations/flameshot.md).

### Prometheus Annotations {#prom-anno}

When `enable_prometheus_annotations: true`, the Operator adds:

```yaml
prometheus.io/scrape: "true"
prometheus.io/port: "8089"
prometheus.io/scheme: "http"
prometheus.io/path: "/metrics"
prometheus.io/param_measurement: "flameshot"
```

The port comes from `FLAMESHOT_HTTP_LOCAL_PORT`. If the Pod already contains any Annotation beginning with `prometheus.io/`, the Operator preserves the user configuration and does not add these Annotations.

## Deployment Example {#flameshot-example}

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

After creation, check:

```shell
kubectl -n production get pod -l app=java-demo -o jsonpath='{.items[0].spec.containers[*].name}'
kubectl -n production logs -l app=java-demo -c datakit-flameshot
```

The result should include `datakit-flameshot`. After Profiling data is generated, view it on the Profiling page in <<<custom_key.brand_name>>>. If there is no data, first check the Sidecar logs, DataKit address, the command regular expression in `processes`, and permissions for the target process.
