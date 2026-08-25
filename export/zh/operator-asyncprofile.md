# 旧版 Java Profiler 注入

该功能通过 `datakit-profiler` Sidecar 运行 async-profiler，属于兼容旧部署的注入方式。新部署建议使用 [Flameshot](operator-flameshot.md)。

## 前置条件 {#async-profiler-prerequisites}

- DataKit 已开启 Profile 采集器。
- 节点允许 `perf_events`，通常要求 `kernel.perf_event_paranoid` 不大于 `2`。
- Pod 安全策略允许 Sidecar 增加 `SYS_PTRACE` 和 `SYS_ADMIN` capability。

## Operator 配置 {#annotation-injection}

当前版本必须先在 `admission_inject_v2.profilers` 中配置匹配规则；只添加 Annotation 不会触发注入。

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

当 `check_annotation: true` 时，Pod 还必须提供 `admission.datakit/java-profiler.version`；其值只替换镜像 tag。`admission.datakit/profiler.enabled: "false"` 可以为单个 Pod 禁用旧版 Profiler。

匹配后，Operator 会添加 `datakit-profiler` Sidecar、共享进程命名空间以及工作目录、`/tmp` 和 `/etc/localtime` 挂载，并将 Pod `restartPolicy` 设置为 `Always`。上线前应确认这些变更符合 Pod 安全策略。

## Deployment 示例 {#async-profiler-example}

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

创建后执行 `kubectl get pod -n production -l app=movies-java -o yaml`，确认存在 `datakit-profiler`。没有数据时检查 Sidecar 日志、内核参数、capability 和 DataKit Profile 接收地址。
