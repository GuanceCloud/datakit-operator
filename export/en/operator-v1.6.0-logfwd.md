# logfwd Usage in v1.6.0 and Earlier

This page is retained only for legacy deployments. For DataKit Operator v1.7.0 and later, use the [current logfwd injection method](operator-logfwd.md) and do not add new static Annotation configurations.

Earlier versions used the `admission.datakit/logfwd.instances` Annotation both to enable the Sidecar and provide collection tasks. Add the Annotation to `.spec.template.metadata.annotations` in the Deployment. Its value is a JSON array string:

```json
[
    {
        "datakit_addr": "datakit-service.datakit.svc:9533",
        "loggings": [
            {
                "logfiles": ["/var/log/app/*.log"],
                "ignore": [],
                "source": "app",
                "service": "checkout",
                "pipeline": "app.p",
                "character_encoding": "",
                "multiline_match": "^\\d{4}-\\d{2}-\\d{2}",
                "tags": {
                    "env": "production"
                }
            }
        ]
    }
]
```

Common fields:

| Field | Description |
| --- | --- |
| `datakit_addr` | DataKit `logfwdserver` address |
| `logfiles` | Array of absolute file paths; glob patterns are supported |
| `ignore` | Array of file paths to exclude; glob patterns are supported |
| `source` | Log source |
| `service` | Service name; `source` is used when this is empty |
| `pipeline` | Pipeline filename on DataKit |
| `character_encoding` | Character encoding; usually left empty for automatic detection |
| `multiline_match` | Regular expression for the first line of a multiline log; backslashes must be escaped in JSON |
| `tags` | Tags added to logs |

## Deployment Example {#datakit-operator-inject-logfwd-example}

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: logging-demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: logging-demo
  template:
    metadata:
      labels:
        app: logging-demo
      annotations:
        admission.datakit/logfwd.instances: '[{"datakit_addr":"datakit-service.datakit.svc:9533","loggings":[{"logfiles":["/var/log/app/*.log"],"source":"app"}]}]'
    spec:
      containers:
        - name: app
          image: busybox:1.36
          command: ["sh", "-c"]
          args:
            - mkdir -p /var/log/app; while true; do date >> /var/log/app/app.log; sleep 1; done
```

After creation, run:

```shell
kubectl get pod -l app=logging-demo -o jsonpath='{.items[0].spec.containers[*].name}'
```

The result should include `datakit-logfwd`. The legacy Operator, YAML, and logfwd image must be used together; when upgrading, migrate all of them to the current CRD-based configuration.
