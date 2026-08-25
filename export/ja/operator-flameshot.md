# DataKit Operator による Flameshot の注入

DataKit Operator は [:octicons-tag-24: v1.8.0](operator-changelog.md#cl-1.8.0) 以降、Flameshot Sidecar の注入をサポートします。Flameshot はスケジュールまたはリソースしきい値に基づいて Java、Python、および Go アプリケーションの Profiling データを収集し、旧方式の Profiler 注入を置き換えます。

## 前提条件 {#flameshot-prerequisites}

- クラスターに [DataKit](https://docs.<<<custom_key.brand_main_domain>>>/datakit/datakit-daemonset-deploy/){:target="_blank"} がインストールされていること。
- DataKit で [Profile コレクター](https://docs.<<<custom_key.brand_main_domain>>>/datakit/datakit-daemonset-deploy/#using-k8-env){:target="_blank"} が有効になっていること。
- 対象 Pod のセキュリティポリシーで、Sidecar への `SYS_PTRACE` capability の追加が許可されていること。
- Prometheus Annotation を有効にする場合は、DataKit で KubernetesPrometheus と Pod Annotation の自動検出も有効にすること。

## Operator の設定 {#flameshot-usage}

`admission_inject_v2.flameshots` にルールを追加します。

```json
{
    "admission_inject_v2": {
        "flameshots": [
            {
                "name": "flameshot-java",
                "namespace_selectors": ["^production$"],
                "label_selectors": ["profiling=flameshot"],
                "image": "{{.FlameshotImage}}",
                "envs": {
                    "FLAMESHOT_DATAKIT_ADDR": "http://datakit-service.datakit:9529/profiling/v1/input",
                    "FLAMESHOT_MONITOR_INTERVAL": "10s",
                    "FLAMESHOT_LOG_LEVEL": "info",
                    "FLAMESHOT_PROFILING_PATH": "/flameshot-data",
                    "FLAMESHOT_LOG_PATH": "/var/log/flameshot.log",
                    "FLAMESHOT_HTTP_LOCAL_IP": "{fieldRef:status.podIP}",
                    "FLAMESHOT_HTTP_LOCAL_PORT": "8089"
                },
                "processes": "[{\"service\":\"java-demo\",\"language\":\"java\",\"command\":\"^java\\\\b.*app\\\\.jar$\",\"events\":\"cpu\",\"duration\":\"30s\",\"cpu_usage_percent\":80}]",
                "enable_prometheus_annotations": true,
                "resources": {
                    "requests": {
                        "cpu": "100m",
                        "memory": "128Mi"
                    },
                    "limits": {
                        "cpu": "200m",
                        "memory": "256Mi"
                    }
                }
            }
        ]
    }
}
```

主なフィールドは次のとおりです。

| フィールド | 説明 |
| --- | --- |
| `name` | ルール名。ログでの特定に使用するため、設定を推奨します |
| `namespace_selectors` | Namespace の正規表現配列 |
| `label_selectors` | Pod Label Selector の配列 |
| `image` | Flameshot Sidecar のイメージ |
| `envs` | Sidecar の環境変数 |
| `processes` | 必須かつ空にできません。プロセスのマッチングと収集ポリシーを指定する JSON 配列文字列 |
| `enable_prometheus_annotations` | Flameshot のメトリクス収集用 Annotation を自動追加するかどうか。デフォルトは `false` |
| `resources` | Sidecar のリソース設定。未指定または無効な場合はデフォルト値を使用します |

`FLAMESHOT_PROFILING_PATH` と有効な `FLAMESHOT_HTTP_LOCAL_PORT` は、注入に必須です。いずれかがない場合や `processes` が空の場合、Operator は注入をスキップして warning を記録します。

Selector と Annotation の共通ルールについては、[DataKit Operator の注入ルール](datakit-operator.md#datakit-operator-inject)を参照してください。`admission.datakit/flameshot.enabled: "false"` を指定すると、Pod 単位で Flameshot を無効にできます。

## 注入結果 {#flameshot-injection-result}

ルールに一致すると、Operator は次の変更を行います。

- `datakit-flameshot` Sidecar を追加し、`SYS_PTRACE` capability を付与します。
- Pod で共有プロセス名前空間を有効にし、Sidecar からアプリケーションプロセスを検出できるようにします。
- `flameshot-volume` EmptyDir を作成し、すべての通常のコンテナの `FLAMESHOT_PROFILING_PATH` にマウントします。
- `processes` を `FLAMESHOT_PROCESSES` として Sidecar に注入します。
- Pod の `restartPolicy` を `Always` に設定します。

Flameshot はアプリケーションプロセスへ直接アクセスします。本番導入前に、Pod Security Admission、コンテナのセキュリティポリシー、およびアプリケーションの実行環境で前述の変更が許可されていることを確認してください。

## 収集設定 {#envs}

主な環境変数は次のとおりです。

| 環境変数 | 説明 |
| --- | --- |
| `FLAMESHOT_DATAKIT_ADDR` | DataKit の Profiling 受信アドレス |
| `FLAMESHOT_MONITOR_INTERVAL` | プロセスとリソースの監視間隔 |
| `FLAMESHOT_LOG_LEVEL` | Flameshot のログレベル |
| `FLAMESHOT_PROFILING_PATH` | Profiling 一時ファイルの共有ディレクトリ。注入に必須 |
| `FLAMESHOT_LOG_PATH` | Flameshot のログパス |
| `FLAMESHOT_HTTP_LOCAL_IP` | Flameshot HTTP のリッスン IP |
| `FLAMESHOT_HTTP_LOCAL_PORT` | Flameshot HTTP とメトリクスのポート。注入に必須 |
| `FLAMESHOT_SERVICE` | すべてのプロセスルールの service を上書き |
| `FLAMESHOT_TAGS` | グローバル Profiling タグ |
| `FLAMESHOT_POD_CPU_LIMIT` | Pod の CPU limit。単位は millicore |
| `FLAMESHOT_POD_MEM_LIMIT` | Pod のメモリー limit。単位は MiB |

`processes` は、Java、Python、および Go のコマンドマッチング、収集時間、CPU/メモリーしきい値、言語固有オプションをサポートします。Heap Dump とオブジェクトストレージへのアップロードも既存の `envs` で設定します。機密性の高い認証情報には `{secretKeyRef:<SECRET>.<KEY>}` の使用を推奨します。全フィールドについては [Flameshot ドキュメント](../integrations/flameshot.md)を参照してください。

### Prometheus Annotation {#prom-anno}

`enable_prometheus_annotations: true` の場合、Operator は次の内容を追加します。

```yaml
prometheus.io/scrape: "true"
prometheus.io/port: "8089"
prometheus.io/scheme: "http"
prometheus.io/path: "/metrics"
prometheus.io/param_measurement: "flameshot"
```

ポートには `FLAMESHOT_HTTP_LOCAL_PORT` の値が使用されます。Pod に `prometheus.io/` で始まる Annotation が1つでも存在する場合、Operator はユーザー設定を維持し、前述の Annotation を追加しません。

## Deployment の例 {#flameshot-example}

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: java-demo
  namespace: production
spec:
  replicas: 1
  selector:
    matchLabels:
      app: java-demo
  template:
    metadata:
      labels:
        app: java-demo
        profiling: flameshot
      annotations:
        admission.datakit/flameshot.enabled: "true"
    spec:
      containers:
        - name: app
          image: example/java-demo:1.0.0
```

作成後、次のコマンドで確認します。

```shell
kubectl -n production get pod -l app=java-demo -o jsonpath='{.items[0].spec.containers[*].name}'
kubectl -n production logs -l app=java-demo -c datakit-flameshot
```

結果に `datakit-flameshot` が含まれている必要があります。Profiling データの生成後は、<<<custom_key.brand_name>>> の Profiling 画面で確認できます。データがない場合は、Sidecar のログ、DataKit のアドレス、`processes` のコマンド正規表現、および対象プロセスの権限を確認してください。
