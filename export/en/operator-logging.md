# Inject Log Configuration with DataKit Operator

DataKit Operator can add the `datakit/logs` Annotation to newly created Pods and create or reuse EmptyDir volumes based on file log paths. This avoids repeatedly maintaining log Annotations and directory mounts in every Deployment.

This feature is intended for DataKit to collect Kubernetes Pod file logs directly. To forward logs to DataKit through a Sidecar, use [logfwd injection](operator-logfwd.md).

## Operator Configuration {#logging-config}

Add a rule to `admission_mutate.loggings`:

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

| Field | Description |
| --- | --- |
| `namespace_selectors` | Array of Namespace regular expressions; at least one matching entry is required |
| `label_selectors` | Array of Pod Label Selectors; at least one matching entry is required |
| `config` | JSON array string written to `datakit/logs` |

Both the Namespace and Label dimensions must match. Multiple selectors within the same dimension are matched with OR; across multiple rules, the first matching entry in configuration order is used.

`config` must be valid JSON. The Operator parses enabled configurations with `type: "file"`, extracts directories from `path`, and mounts them. A `stdout` configuration is written only to the Annotation and does not require a new volume.

## Injection Result {#logging-injection-result}

For a Pod matching the preceding example, the Operator adds:

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

The Operator mounts the directory into all regular application containers:

- If an EmptyDir is already mounted at the path, the existing volume is reused.
- If the path has no corresponding mount, a new EmptyDir is created.
- If another volume type is already mounted at the path, a warning is logged and that path is skipped.
- If the Pod already has a `datakit/logs` Annotation, the user configuration is preserved and neither the Annotation nor volumes are modified.

## Deployment Example {#logging-example}

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

After creating the Pod, check it with:

```shell
kubectl -n middleware get pod -l app=logging -o yaml
kubectl logs -n datakit deployment/datakit-operator
```

The resulting Pod should contain the `datakit/logs` Annotation and the EmptyDir and mount corresponding to `/var/log/app`. Recreate the Pod after changing a rule for the change to take effect.
