# Legacy Python Profiler Injection

This feature runs py-spy through the `datakit-profiler` Sidecar and is retained for compatibility with legacy deployments. It supports only CPython. [Flameshot](operator-flameshot.md) is recommended for new deployments.

## Operator Configuration {#prerequisites}

The current version requires a matching rule in `admission_inject_v2.profilers`; adding only an Annotation does not trigger injection.

```json
{
    "admission_inject_v2": {
        "profilers": [
            {
                "name": "legacy-python-profiler",
                "language": "python",
                "namespace_selectors": ["^production$"],
                "label_selectors": ["profiling=py-spy"],
                "check_annotation": false,
                "image": "{{.K8sProfilersPySpyImage}}",
                "envs": {
                    "DK_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DK_AGENT_PORT": "9529",
                    "DK_PROFILE_DURATION": "240",
                    "DK_PROFILE_SCHEDULE": "0 * * * *"
                }
            }
        ]
    }
}
```

When `check_annotation: true`, the Pod must also provide `admission.datakit/python-profiler.version`; its value replaces only the image tag. `admission.datakit/profiler.enabled: "false"` disables the legacy Profiler for an individual Pod.

After a match, the Operator adds the `datakit-profiler` Sidecar, a shared process namespace, and mounts for the working directory, `/tmp`, and `/etc/localtime`. It also sets the Pod `restartPolicy` to `Always`. The Sidecar adds the `SYS_PTRACE` and `SYS_ADMIN` capabilities, so confirm that the Pod security policy allows them before use.

## Deployment Example {#pyspy-example}

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: movies-python
  namespace: production
spec:
  replicas: 1
  selector:
    matchLabels:
      app: movies-python
  template:
    metadata:
      labels:
        app: movies-python
        profiling: py-spy
    spec:
      containers:
        - name: app
          image: example/movies-python:1.2.3
```

After creation, run `kubectl get pod -n production -l app=movies-python -o yaml` and confirm that `datakit-profiler` is present. If no data is available, check the Sidecar logs, capabilities, whether the target process is CPython, and the DataKit Profile receiver address.
