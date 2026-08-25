# Injecting DDTrace with DataKit Operator

DataKit Operator injects DDTrace automatic instrumentation when a Pod is created. It supports Java, Python, PHP, and Node.js. The Operator adds a `datakit-lib-init` init Container and a shared `/datadog-lib` volume to the Pod, then modifies the startup environment of all regular application containers. Existing Pods are not modified and must be recreated for injection to take effect.

## Usage {#datakit-operator-inject-lib-usage}

Before use, [install DataKit Operator](datakit-operator.md#install) and confirm that application containers can access the DataKit Trace receiver address.

### Supported Runtimes and Images {#ddtrace-lib-image-selection}

| Language | Application runtime | Default image |
| --- | --- | --- |
| Java | JVM supported by the DDTrace Java Agent | `{{.DDTraceJavaImage}}` |
| Python | Python 3.7 | `{{.DDTracePython37Image}}` |
| Python | Python 3.8 | `{{.DDTracePython38Image}}` |
| Python | Python 3.9 to 3.14 | `{{.DDTracePythonImage}}` |
| PHP | Linux GNU or musl | `{{.DDTracePHPImage}}` |
| Node.js | Node.js 16 | `{{.DDTraceNodeJS16Image}}` |
| Node.js | Node.js 18 to 25 | `{{.DDTraceNodeJSImage}}` |

The image version must be compatible with the language runtime in the application container. In offline environments, synchronize the image to a private registry and specify its full address in the rule's `image` field.

PHP rules also require `php_loader_flavor`: use `linux-gnu` for glibc images and `linux-musl` for musl images such as Alpine. The Operator does not automatically detect which libc the application image uses. A missing or invalid value falls back to `linux-gnu`.

The PHP init Container copies the loader configuration for the corresponding libc. After the application process starts, the Datadog loader selects a compatible `.so` file based on the PHP version, ABI, and ZTS/NTS mode. The Operator itself selects only the libc type and does not inspect the PHP runtime.

### Rule Configuration {#ddtrace-config}

Add rules to `admission_inject_v2.ddtraces`. The following Java rule matches only Pods in the `default` Namespace with the `admission.datakit/ddtrace-language=java` label:

```json
{
    "name": "ddtrace-java",
    "language": "java",
    "namespace_selectors": ["^default$"],
    "label_selectors": ["admission.datakit/ddtrace-language=java"],
    "check_annotation": false,
    "image": "{{.DDTraceJavaImage}}",
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
```

Common fields:

| Field | Description |
| --- | --- |
| `name` | Rule name, recommended for locating relevant logs |
| `language` | Required; one of `java`, `python`, `php`, or `nodejs` |
| `namespace_selectors` | Array of Namespace regular expressions |
| `label_selectors` | Array of Pod Label Selectors |
| `check_annotation` | Whether the Pod must provide a version Annotation for the corresponding language; defaults to `false` |
| `image` | Required; image for the language library init Container |
| `envs` | Environment variables injected into all regular application containers |
| `resources` | Resource configuration for the init Container; defaults are used when missing or invalid |
| `php_loader_flavor` | PHP only; `linux-gnu` or `linux-musl` |

For the general rules governing selectors, Annotations, default resources, and environment variable references, see [DataKit Operator injection rules](datakit-operator.md#datakit-operator-inject). Use mutually exclusive labels for rules targeting different languages so that a Pod does not match multiple rules.

To configure Python, PHP, or Node.js, change `name`, `language`, `label_selectors`, and `image` in the preceding example. For PHP, also add `php_loader_flavor`. For example:

```json
{
    "name": "ddtrace-php-musl",
    "language": "php",
    "namespace_selectors": ["^default$"],
    "label_selectors": ["admission.datakit/ddtrace-language=php"],
    "check_annotation": false,
    "image": "{{.DDTracePHPImage}}",
    "php_loader_flavor": "linux-musl",
    "envs": {
        "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
        "DD_TRACE_AGENT_PORT": "9529",
        "DD_SERVICE": "{fieldRef:metadata.labels['app']}"
    }
}
```

## Injection Method {#ddtrace-injection}

| Language | Operator changes to application containers |
| --- | --- |
| Java | Mounts `/datadog-lib` and appends to `JAVA_TOOL_OPTIONS` the value `-javaagent:/datadog-lib/dd-java-agent.jar` |
| Python | Mounts `/datadog-lib` and prepends `/datadog-lib/` to `PYTHONPATH` |
| PHP | Mounts `/datadog-lib`, configures `DD_LOADER_PACKAGE_PATH` and `PHP_INI_SCAN_DIR`, and loads `dd_library_loader.ini` |
| Node.js | Mounts `/datadog-lib` and appends to `NODE_OPTIONS` the value `--require=/datadog-lib/node_modules/dd-trace/init` |

The Operator preserves existing startup arguments with the same names in application containers. If `JAVA_TOOL_OPTIONS`, `PYTHONPATH`, `PHP_INI_SCAN_DIR`, or `NODE_OPTIONS` uses Kubernetes `valueFrom`, the Operator cannot merge strings safely. It skips the entire DDTrace injection and logs a warning to avoid leaving behind an unusable init Container.

Ordinary environment variables in a rule do not overwrite variables with the same name in an application container. `DD_TAGS` is an exception: when both values are ordinary strings, the Operator merges the tags.

## Annotations and Versions {#check-annotation-config}

`admission.datakit/ddtrace.enabled: "false"` disables DDTrace for an individual Pod. This switch always applies, regardless of `check_annotation`.

When a rule sets `check_annotation: true`, the Pod must also provide the version Annotation for the corresponding language:

| Language | Version Annotation |
| --- | --- |
| Java | `admission.datakit/java-lib.version` |
| Python | `admission.datakit/python-lib.version` |
| PHP | `admission.datakit/php-lib.version` |
| Node.js | `admission.datakit/nodejs-lib.version` |

The version value replaces only the tag in the rule's `image`; it does not change the image registry or name. It can therefore switch only between versions of the same image.

If multiple DDTrace rules match, the Operator uses the first rule in configuration order that satisfies the Annotation conditions. When both DDTrace and OpenTelemetry match, DDTrace takes precedence. If DDTrace is selected but injection fails, the Operator does not fall back to OpenTelemetry. Do not enable both automatic instrumentation libraries for the same Pod.

## Deployment Example {#anno-demo}

The following Deployment matches the preceding Java rule:

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

To explicitly disable DDTrace, change the Annotation to:

```yaml
admission.datakit/ddtrace.enabled: "false"
```

## Verification and Troubleshooting {#ddtrace-verify}

After recreating the Pod, inspect the resulting Pod:

```shell
kubectl get pod <pod-name> -o jsonpath='{.spec.initContainers[*].name}'
kubectl get pod <pod-name> -o yaml
kubectl logs -n datakit deployment/datakit-operator
```

The Pod should contain `datakit-lib-init`, the `datakit-auto-instrument` volume, and the mounts and startup environment for the corresponding language. Then send an actual request and confirm the Trace data in the DataKit monitor or on the <<<custom_key.brand_name>>> page.

Common issues:

- A running Pod did not change: recreate it; the Operator processes only `CREATE`.
- Nothing was injected: confirm that the Namespace, Label, and `check_annotation` conditions all match.
- The init Container cannot pull its image: check the image address and credentials in the rule, and cluster network connectivity.
- The init Container succeeds but no Trace appears: confirm that the application process preserves the injected startup environment and can reach `DD_AGENT_HOST` and `DD_TRACE_AGENT_PORT`.
- PHP fails to start: confirm whether the application image uses glibc or musl and set the correct `php_loader_flavor`.
