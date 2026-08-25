# Operator < 1.6.0 での logfwd の使用方法

この設定方法は DataKit-Operator v1.6.0 以前のバージョンで使用します。v1.7.0 では新しい CRD 設定方式が採用され、v1.6.0 で導入された CRD + Annotation のハイブリッド方式は廃止されました。

1. 対象の Kubernetes クラスターで、[DataKit-Operator をダウンロードしてインストール](datakit-operator.md#install)します
1. Deployment に指定の Annotation を追加し、logfwd sidecar をマウントすることを指定します。Annotation は template 内に追加してください
    - key はすべて `admission.datakit/logfwd.instances` です
    - value は具体的な logfwd 設定を含む JSON 文字列です。例は次のとおりです：

```json
[
    {
        "datakit_addr": "datakit-service.datakit.svc:9533",
        "loggings": [
            {
                "logfiles":      ["<your-logfile-path>"],
                "ignore":        [],
                "storage_index": "<your-storage-index>",
                "source":        "<your-source>",
                "service":       "<your-service>",
                "pipeline":      "<your-pipeline.p>",
                "character_encoding": "",
                "multiline_match": "<your-match>",
                "tags": {}
            },
            {
                "logfiles": ["<your-logfile-path-2>"],
                "source": "<your-source-2>"
            }
        ]
    }
]
```

パラメーターについては、[logfwd の設定](../integrations/logfwd.md#config)を参照してください：

- `datakit_addr` は DataKit logfwdserver のアドレスです
- `loggings` は主要な設定であり、配列です。[DataKit logging コレクター](../integrations/logging.md)を参照してください
    - `logfiles` ログファイルの一覧です。絶対パスを指定でき、glob ルールによる一括指定をサポートします。絶対パスの使用を推奨します
    - `ignore` ファイルパスのフィルターです。glob ルールを使用し、いずれかのフィルター条件に一致するファイルは収集されません
    - `storage_index` ログの保存先インデックスを指定します
    - `source` データソースです。空の場合、デフォルトで 'default' を使用します
    - `service` tag を追加します。空の場合、デフォルトで $source を使用します
    - `pipeline` Pipeline スクリプトのパスです。空の場合は $source.p を使用し、$source.p が存在しない場合は Pipeline を使用しません（このスクリプトファイルは DataKit 側に存在します）
    - `character_encoding` エンコーディングを選択します。誤ったエンコーディングを指定するとデータを表示できなくなるため、通常は空のままにしてください。`utf-8/utf-16le/utf-16le/gbk/gb18030` をサポートします
    - `multiline_match` 複数行マッチングです。詳細については、[DataKit ログの複数行設定](../integrations/logging.md#multiline)を参照してください。JSON 形式のため、3 つのシングルクォートを使用する「エスケープしない記述方法」はサポートされません。正規表現 `^\d{4}` は、エスケープを追加して `^\\d{4}` と記述する必要があります
    - `tags` 追加の `tag` を追加します。JSON map 形式で記述します。例：`{ "key1":"value1", "key2":"value2" }`

<!-- markdownlint-disable MD046 -->
???+ note

    logfwd の注入時、DataKit Operator はデフォルトで同じパスの volume を再利用し、同じパスの volume が存在することによる注入エラーを回避します。

    パス末尾のスラッシュの有無によって意味が異なります。たとえば、`/var/log` と `/var/log/` は異なるパスであるため、再利用できません。
<!-- markdownlint-enable MD046 -->

## ユースケース {#datakit-operator-inject-logfwd-example}

以下は、shell を使用してファイルに継続的にデータを書き込み、そのファイルを収集対象として設定する Deployment の例です：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
    name: logging-deployment
    labels:
    app: logging
spec:
    replicas: 1
    selector:
    matchLabels:
        app: logging
    template:
    metadata:
        labels:
        app: logging
        annotations:
        admission.datakit/logfwd.instances: '[{"datakit_addr":"datakit-service.datakit.svc:9533","loggings":[{"logfiles":["/var/log/log-test/*.log"],"source":"deployment-logging","tags":{"key01":"value01"}}]}]'
    spec:
        containers:
        - name: log-container
        image: busybox
        args: [/bin/sh, -c, 'mkdir -p /var/log/log-test; i=0; while true; do printf "$(date "+%F %H:%M:%S") [%-8d] Bash For Loop Examples.\\n" $i >> /var/log/log-test/1.log; i=$((i+1)); sleep 1; done']
```

YAML ファイルを使用してリソースを作成します：

```shell
$ kubectl apply -f logging.yaml
...
```

次のように確認します：

```shell
$ kubectl get pod

NAME                                   READY   STATUS    RESTARTS      AGE
logging-deployment-5d48bf9995-vt6bb    1/1     Running   0             4s

$ kubectl get pod logging-deployment-5d48bf9995-vt6bb -o=jsonpath={.spec.containers\[\*\].name}
log-container datakit-logfwd
```

最後に、<<<custom_key.brand_name>>>ログプラットフォームでログが収集されているかどうかを確認できます。
