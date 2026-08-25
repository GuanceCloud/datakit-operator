# Legacy Java Profiler Injection

This feature runs async-profiler through the `datakit-profiler` Sidecar and is retained for compatibility with legacy deployments. [Flameshot](operator-flameshot.md) is recommended for new deployments.

## Prerequisites {#async-profiler-prerequisites}

- The Profile collector is enabled in DataKit.
- Nodes allow `perf_events`; this usually requires `kernel.perf_event_paranoid` to be no greater than `2`.
- Pod security policies allow the Sidecar to add the `SYS_PTRACE` and `SYS_ADMIN` capabilities.

## Operator Configuration {#annotation-injection}

The current version requires a matching rule in `admission_inject_v2.profilers`; adding only an Annotation does not trigger injection.

```json
{
    "admission_inject_v2": {
        "profilers": [
            {
                "name": "legacy-java-profiler",
                "language": "java",
                "namespace_selectors": ["^production$"],
                "label_selectors": ["profiling=async-profiler"],
                "check_annotation": false,
                "image": "{{.K8sProfilersAsyncProfileImage}}",
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

When `check_annotation: true`, the Pod must also provide `admission.datakit/java-profiler.version`; its value replaces only the image tag. `admission.datakit/profiler.enabled: "false"` disables the legacy Profiler for an individual Pod.

After a match, the Operator adds the `datakit-profiler` Sidecar, a shared process namespace, and mounts for the working directory, `/tmp`, and `/etc/localtime`. It also sets the Pod `restartPolicy` to `Always`. Before deployment, confirm that these changes comply with the Pod security policy.

## Deployment Example {#async-profiler-example}

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: movies-java
  namespace: production
spec:
  replicas: 1
  selector:
    matchLabels:
      app: movies-java
  template:
    metadata:
      labels:
        app: movies-java
        profiling: async-profiler
    spec:
      containers:
        - name: app
          image: example/movies-java:1.2.3
          securityContext:
            seccompProfile:
              type: Unconfined
```

After creation, run `kubectl get pod -n production -l app=movies-java -o yaml` and confirm that `datakit-profiler` is present. If no data is available, check the Sidecar logs, kernel settings, capabilities, and the DataKit Profile receiver address.
