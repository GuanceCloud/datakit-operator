# 旧方式の Java Profiler 注入

この機能は `datakit-profiler` Sidecar で async-profiler を実行する、既存デプロイとの互換性を維持するための注入方式です。新規デプロイでは [Flameshot](operator-flameshot.md) の使用を推奨します。

## 前提条件 {#async-profiler-prerequisites}

- DataKit で Profile コレクターが有効になっていること。
- ノードで `perf_events` が許可されていること。通常は `kernel.perf_event_paranoid` が `2` 以下である必要があります。
- Pod のセキュリティポリシーで、Sidecar への `SYS_PTRACE` および `SYS_ADMIN` capability の追加が許可されていること。

## Operator の設定 {#annotation-injection}

現行バージョンでは、先に `admission_inject_v2.profilers` へマッチングルールを設定する必要があります。Annotation を追加するだけでは注入されません。

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

`check_annotation: true` の場合、Pod に `admission.datakit/java-profiler.version` も指定する必要があります。この値で置き換わるのはイメージの tag だけです。`admission.datakit/profiler.enabled: "false"` を指定すると、Pod 単位で旧方式の Profiler を無効にできます。

ルールに一致すると、Operator は `datakit-profiler` Sidecar、共有プロセス名前空間、作業ディレクトリ、`/tmp` および `/etc/localtime` のマウントを追加し、Pod の `restartPolicy` を `Always` に設定します。本番導入前に、これらの変更が Pod のセキュリティポリシーに適合することを確認してください。

## Deployment の例 {#async-profiler-example}

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

作成後、`kubectl get pod -n production -l app=movies-java -o yaml` を実行し、`datakit-profiler` が存在することを確認します。データがない場合は、Sidecar のログ、カーネルパラメーター、capability、および DataKit の Profile 受信アドレスを確認してください。
