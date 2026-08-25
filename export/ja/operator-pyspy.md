# 旧方式の Python Profiler 注入

この機能は `datakit-profiler` Sidecar で py-spy を実行する、既存デプロイとの互換性を維持するための注入方式で、CPython のみをサポートします。新規デプロイでは [Flameshot](operator-flameshot.md) の使用を推奨します。

## Operator の設定 {#prerequisites}

現行バージョンでは、先に `admission_inject_v2.profilers` へマッチングルールを設定する必要があります。Annotation を追加するだけでは注入されません。

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

`check_annotation: true` の場合、Pod に `admission.datakit/python-profiler.version` も指定する必要があります。この値で置き換わるのはイメージの tag だけです。`admission.datakit/profiler.enabled: "false"` を指定すると、Pod 単位で旧方式の Profiler を無効にできます。

ルールに一致すると、Operator は `datakit-profiler` Sidecar、共有プロセス名前空間、作業ディレクトリ、`/tmp` および `/etc/localtime` のマウントを追加し、Pod の `restartPolicy` を `Always` に設定します。Sidecar には `SYS_PTRACE` および `SYS_ADMIN` capability が追加されるため、使用前に Pod のセキュリティポリシーで許可されていることを確認してください。

## Deployment の例 {#pyspy-example}

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

作成後、`kubectl get pod -n production -l app=movies-python -o yaml` を実行し、`datakit-profiler` が存在することを確認します。データがない場合は、Sidecar のログ、capability、対象プロセスが CPython であること、および DataKit の Profile 受信アドレスを確認してください。
