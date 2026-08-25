# v1.6.0 及以前的 logfwd 用法

本页仅用于维护旧部署。DataKit Operator v1.7.0 及以后版本请使用 [当前 logfwd 注入方式](operator-logfwd.md)，不要继续新增 Annotation 静态配置。

旧版本通过 `admission.datakit/logfwd.instances` Annotation 同时启用 Sidecar 并提供采集任务。Annotation 必须添加到 Deployment 的 `.spec.template.metadata.annotations`，值是 JSON 数组字符串：

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

常用字段：

| 字段 | 说明 |
| --- | --- |
| `datakit_addr` | DataKit `logfwdserver` 地址 |
| `logfiles` | 文件绝对路径数组，支持 glob |
| `ignore` | 需要排除的文件路径数组，支持 glob |
| `source` | 日志来源 |
| `service` | 服务名；为空时使用 `source` |
| `pipeline` | DataKit 端的 Pipeline 文件名 |
| `character_encoding` | 字符编码；通常留空自动检测 |
| `multiline_match` | 多行首行正则；JSON 中的反斜线需要转义 |
| `tags` | 附加到日志的标签 |

## Deployment 示例 {#datakit-operator-inject-logfwd-example}

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

创建后执行：

```shell
kubectl get pod -l app=logging-demo -o jsonpath='{.items[0].spec.containers[*].name}'
```

结果应包含 `datakit-logfwd`。旧版 Operator、YAML 和 logfwd 镜像需要配套使用；升级时应同时迁移到当前 CRD 配置方式。
