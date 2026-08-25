# DataKit Operator によるログ収集設定の注入

DataKit Operator は、新規 Pod に `datakit/logs` Annotation を追加し、ファイルログのパスに基づいて EmptyDir ボリュームを作成または再利用できます。これにより、各 Deployment でログ Annotation とディレクトリのマウントを重複して管理する必要がなくなります。

この機能は、DataKit が Kubernetes Pod のファイルログを直接収集する場合に使用します。Sidecar から DataKit へログを転送する場合は、[logfwd の注入](operator-logfwd.md)を使用してください。

## Operator の設定 {#logging-config}

`admission_mutate.loggings` にルールを追加します。

```json
{
    "admission_mutate": {
        "loggings": [
            {
                "namespace_selectors": ["^middleware$"],
                "label_selectors": ["app=logging"],
                "config": "[{\"disable\":false,\"type\":\"file\",\"path\":\"/var/log/app/*.log\",\"source\":\"logging-demo\"}]"
            }
        ]
    }
}
```

| フィールド | 説明 |
| --- | --- |
| `namespace_selectors` | Namespace の正規表現配列。マッチ可能な項目を1つ以上指定する必要があります |
| `label_selectors` | Pod Label Selector の配列。マッチ可能な項目を1つ以上指定する必要があります |
| `config` | `datakit/logs` に書き込む JSON 配列文字列 |

Namespace と Label の両方が一致する必要があります。同じ次元内の複数の selector は OR 条件で評価され、複数のルールでは設定順に最初に一致したものが使用されます。

`config` は有効な JSON である必要があります。Operator は、無効化されていない `type: "file"` 設定を解析し、`path` からディレクトリを抽出してマウントします。`stdout` 設定は Annotation に書き込まれるだけで、新しいボリュームは追加されません。

## 注入結果 {#logging-injection-result}

前述の例に一致する Pod には、Operator により次の内容が追加されます。

```yaml
metadata:
  annotations:
    datakit/logs: '[{"disable":false,"type":"file","path":"/var/log/app/*.log","source":"logging-demo"}]'
spec:
  containers:
    - name: app
      volumeMounts:
        - name: datakit-logs-volume-0
          mountPath: /var/log/app
  volumes:
    - name: datakit-logs-volume-0
      emptyDir: {}
```

Operator は、すべての通常のアプリケーションコンテナにディレクトリをマウントします。

- 対象パスに EmptyDir がすでにマウントされている場合は、既存のボリュームを再利用します。
- 対応するマウントがない場合は、新しい EmptyDir を作成します。
- 対象パスで別の種類のボリュームが使用されている場合は、warning を記録してそのパスをスキップします。
- Pod に `datakit/logs` Annotation がすでにある場合は、ユーザー設定を維持し、Annotation もボリュームも変更しません。

## Deployment の例 {#logging-example}

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: logging-demo
  namespace: middleware
spec:
  replicas: 1
  selector:
    matchLabels:
      app: logging
  template:
    metadata:
      labels:
        app: logging
    spec:
      containers:
        - name: app
          image: nginx:1.25
```

Pod の作成後、次のコマンドで確認します。

```shell
kubectl -n middleware get pod -l app=logging -o yaml
kubectl logs -n datakit deployment/datakit-operator
```

最終的な Pod には、`datakit/logs` Annotation と、`/var/log/app` に対応する EmptyDir およびマウントが含まれます。ルールの変更を反映するには、Pod を再作成する必要があります。
