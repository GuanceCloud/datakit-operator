# 旧版 Python Profiler 注入

该功能通过 `datakit-profiler` Sidecar 运行 py-spy，属于兼容旧部署的注入方式，仅支持 CPython。新部署建议使用 [Flameshot](operator-flameshot.md)。

## Operator 配置 {#prerequisites}

当前版本必须先在 `admission_inject_v2.profilers` 中配置匹配规则；只添加 Annotation 不会触发注入。

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

当 `check_annotation: true` 时，Pod 还必须提供 `admission.datakit/python-profiler.version`；其值只替换镜像 tag。`admission.datakit/profiler.enabled: "false"` 可以为单个 Pod 禁用旧版 Profiler。

匹配后，Operator 会添加 `datakit-profiler` Sidecar、共享进程命名空间以及工作目录、`/tmp` 和 `/etc/localtime` 挂载，并将 Pod `restartPolicy` 设置为 `Always`。Sidecar 会增加 `SYS_PTRACE` 和 `SYS_ADMIN` capability，使用前应确认 Pod 安全策略允许。

## Deployment 示例 {#pyspy-example}

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

创建后执行 `kubectl get pod -n production -l app=movies-python -o yaml`，确认存在 `datakit-profiler`。没有数据时检查 Sidecar 日志、capability、目标进程是否为 CPython，以及 DataKit Profile 接收地址。
