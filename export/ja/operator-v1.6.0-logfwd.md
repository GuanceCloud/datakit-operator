# v1.6.0 以前の logfwd の使用方法

このページは既存デプロイの保守のみを対象としています。DataKit Operator v1.7.0 以降では[現行の logfwd 注入方式](operator-logfwd.md)を使用し、Annotation による静的設定を新たに追加しないでください。

旧バージョンでは、`admission.datakit/logfwd.instances` Annotation で Sidecar を有効にすると同時に収集タスクを指定します。Annotation は Deployment の `.spec.template.metadata.annotations` に追加し、値には JSON 配列文字列を指定します。

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

主なフィールドは次のとおりです。

| フィールド | 説明 |
| --- | --- |
| `datakit_addr` | DataKit `logfwdserver` のアドレス |
| `logfiles` | ファイルの絶対パスの配列。glob を使用できます |
| `ignore` | 除外するファイルパスの配列。glob を使用できます |
| `source` | ログのソース |
| `service` | サービス名。空の場合は `source` を使用します |
| `pipeline` | DataKit 側の Pipeline ファイル名 |
| `character_encoding` | 文字エンコーディング。通常は空にして自動検出します |
| `multiline_match` | 複数行ログの先頭行を判定する正規表現。JSON 内ではバックスラッシュをエスケープする必要があります |
| `tags` | ログに追加するタグ |

## Deployment の例 {#datakit-operator-inject-logfwd-example}

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

作成後、次のコマンドを実行します。

```shell
kubectl get pod -l app=logging-demo -o jsonpath='{.items[0].spec.containers[*].name}'
```

結果に `datakit-logfwd` が含まれている必要があります。旧版の Operator、YAML、および logfwd イメージは対応する組み合わせで使用してください。アップグレード時には、現行の CRD 設定方式へ同時に移行してください。
