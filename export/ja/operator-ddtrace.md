# DataKit Operator による DDTrace の注入

DataKit Operator は Pod の作成時に DDTrace 自動計装を注入し、Java、Python、PHP、および Node.js をサポートします。Operator は Pod に `datakit-lib-init` init Container と共有ボリューム `/datadog-lib` を追加し、すべての通常のアプリケーションコンテナの起動環境を変更します。既存の Pod は変更されないため、反映するには再作成が必要です。

## 使用方法 {#datakit-operator-inject-lib-usage}

使用前に [DataKit Operator をインストール](datakit-operator.md#install)し、アプリケーションコンテナから DataKit の Trace 受信アドレスへアクセスできることを確認してください。

配布テンプレートでは、デフォルトで Java DDTrace ルールを 1 つだけ残し、`default` Namespace のすべての Pod に一致させています。これは既存のデプロイとの互換性を維持するためであり、Java だけをサポートするという意味ではありません。Operator はアプリケーションコンテナの言語を自動検出しません。Python、PHP、または Node.js を設定する場合は、デフォルトの Java ルールも相互排他的な言語ラベルを使用するように変更してください。変更しない場合、そのルールが先に Pod に一致し、後続の言語ルールが有効になりません。

### サポート範囲とイメージ {#ddtrace-lib-image-selection}

| 言語 | アプリケーションランタイム | デフォルトイメージ |
| --- | --- | --- |
| Java | DDTrace Java Agent が対応する JVM | `{{.DDTraceJavaImage}}` |
| Python | Python 3.7 | `{{.DDTracePython37Image}}` |
| Python | Python 3.8 | `{{.DDTracePython38Image}}` |
| Python | Python 3.9～3.14 | `{{.DDTracePythonImage}}` |
| PHP | Linux GNU または musl | `{{.DDTracePHPImage}}` |
| Node.js | Node.js 16 | `{{.DDTraceNodeJS16Image}}` |
| Node.js | Node.js 18～25 | `{{.DDTraceNodeJSImage}}` |

イメージのバージョンは、アプリケーションコンテナの言語ランタイムと互換性がある必要があります。オフライン環境ではイメージをプライベートレジストリへ同期し、ルールの `image` に完全なアドレスを指定できます。

PHP ルールでは `php_loader_flavor` も設定する必要があります。glibc イメージには `linux-gnu`、Alpine などの musl イメージには `linux-musl` を指定します。Operator はアプリケーションイメージの libc を自動検出しません。未設定または設定が正しくない場合は `linux-gnu` にフォールバックします。

PHP init Container は、対応する libc の loader 設定をコピーします。アプリケーションプロセスの起動後、Datadog loader が PHP バージョン、ABI、および ZTS/NTS モードに基づいて互換性のある `.so` ファイルを読み込みます。Operator 自体は libc の種類だけを選択し、PHP ランタイムは検査しません。

### ルールの設定 {#ddtrace-config}

次の設定には Java、Python、PHP、および Node.js のルールがすべて含まれています。既存の `jsonconfig` を編集する場合は、この `ddtraces` 配列全体で `admission_inject_v2.ddtraces` を置き換え、`admission_inject_v2` 配下のその他の設定は維持してください。配布テンプレートのデフォルト Java ルールの後ろにこれらのルールをそのまま追加しないでください。デフォルトルールが先に一致します。

すべてのルールで `"namespace_selectors": ["*"]` を使用しますが、対応する言語ラベルを持つ Pod だけが一致します。そのため、ラベルのない Pod に自動注入することなく、Namespace をまたいで利用できます。サンプルの Python イメージは Python 3.9～3.14、PHP は `linux-gnu`、Node.js イメージは Node.js 18～25 向けです。その他のランタイムでは、前の表に従ってイメージまたは `php_loader_flavor` を変更してください。

```json
{
    "admission_inject_v2": {
        "ddtraces": [
            {
                "name": "ddtrace-java",
                "language": "java",
                "namespace_selectors": ["*"],
                "label_selectors": ["admission.datakit/ddtrace-language=java"],
                "check_annotation": false,
                "image": "{{.DDTraceJavaImage}}",
                "envs": {
                    "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DD_TRACE_AGENT_PORT": "9529",
                    "DD_JMXFETCH_STATSD_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DD_JMXFETCH_STATSD_PORT": "8125",
                    "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
                },
                "resources": {
                    "requests": {
                        "cpu": "100m",
                        "memory": "64Mi"
                    },
                    "limits": {
                        "cpu": "500m",
                        "memory": "512Mi"
                    }
                }
            },
            {
                "name": "ddtrace-python",
                "language": "python",
                "namespace_selectors": ["*"],
                "label_selectors": ["admission.datakit/ddtrace-language=python"],
                "check_annotation": false,
                "image": "{{.DDTracePythonImage}}",
                "envs": {
                    "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DD_TRACE_AGENT_PORT": "9529",
                    "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
                },
                "resources": {
                    "requests": {
                        "cpu": "100m",
                        "memory": "64Mi"
                    },
                    "limits": {
                        "cpu": "500m",
                        "memory": "512Mi"
                    }
                }
            },
            {
                "name": "ddtrace-php",
                "language": "php",
                "php_loader_flavor": "linux-gnu",
                "namespace_selectors": ["*"],
                "label_selectors": ["admission.datakit/ddtrace-language=php"],
                "check_annotation": false,
                "image": "{{.DDTracePHPImage}}",
                "envs": {
                    "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DD_TRACE_AGENT_PORT": "9529",
                    "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
                },
                "resources": {
                    "requests": {
                        "cpu": "100m",
                        "memory": "64Mi"
                    },
                    "limits": {
                        "cpu": "500m",
                        "memory": "512Mi"
                    }
                }
            },
            {
                "name": "ddtrace-nodejs",
                "language": "nodejs",
                "namespace_selectors": ["*"],
                "label_selectors": ["admission.datakit/ddtrace-language=nodejs"],
                "check_annotation": false,
                "image": "{{.DDTraceNodeJSImage}}",
                "envs": {
                    "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DD_TRACE_AGENT_PORT": "9529",
                    "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
                },
                "resources": {
                    "requests": {
                        "cpu": "100m",
                        "memory": "64Mi"
                    },
                    "limits": {
                        "cpu": "500m",
                        "memory": "512Mi"
                    }
                }
            }
        ]
    }
}
```

Pod には対応する言語ラベルを 1 つだけ設定します。例：

```yaml
metadata:
  labels:
    app: payment-service
    admission.datakit/ddtrace-language: python
```

主なフィールドは次のとおりです。

| フィールド | 説明 |
| --- | --- |
| `name` | ルール名。ログでの特定に使用するため、設定を推奨します |
| `language` | 必須。`java`、`python`、`php`、または `nodejs` を指定できます |
| `namespace_selectors` | Namespace の正規表現配列 |
| `label_selectors` | Pod Label Selector の配列 |
| `check_annotation` | 対応する言語のバージョン Annotation を Pod に要求するかどうか。デフォルトは `false` |
| `image` | 必須。言語ライブラリ用 init Container のイメージ |
| `envs` | すべての通常のアプリケーションコンテナに注入する環境変数 |
| `resources` | init Container のリソース設定。未指定または無効な場合はデフォルト値を使用します |
| `php_loader_flavor` | PHP だけで使用。`linux-gnu` または `linux-musl` を指定できます |

Selector、Annotation、デフォルトリソース、および環境変数参照の共通ルールについては、[DataKit Operator の注入ルール](datakit-operator.md#datakit-operator-inject)を参照してください。複数言語のルールには相互排他的なラベルを使用し、1つの Pod が複数のルールに一致しないようにしてください。

## 注入方法 {#ddtrace-injection}

| 言語 | Operator によるアプリケーションコンテナの変更 |
| --- | --- |
| Java | `/datadog-lib` をマウントし、`JAVA_TOOL_OPTIONS` に `-javaagent:/datadog-lib/dd-java-agent.jar` を追加 |
| Python | `/datadog-lib` をマウントし、`/datadog-lib/` を `PYTHONPATH` の先頭に追加 |
| PHP | `/datadog-lib` をマウントし、`DD_LOADER_PACKAGE_PATH` と `PHP_INI_SCAN_DIR` を設定して `dd_library_loader.ini` を読み込み |
| Node.js | `/datadog-lib` をマウントし、`NODE_OPTIONS` に `--require=/datadog-lib/node_modules/dd-trace/init` を追加 |

Operator は、アプリケーションコンテナにある同名の起動パラメーターを維持します。`JAVA_TOOL_OPTIONS`、`PYTHONPATH`、`PHP_INI_SCAN_DIR`、または `NODE_OPTIONS` が Kubernetes の `valueFrom` を使用している場合、文字列を安全にマージできないため、Operator は DDTrace の注入全体をスキップして warning を記録します。これにより、無効な init Container だけが残ることを防ぎます。

ルール内の通常の環境変数は、アプリケーションコンテナにある同名の変数を上書きしません。`DD_TAGS` は例外で、両方が通常の文字列の場合は Operator がタグをマージします。

## Annotation とバージョン {#check-annotation-config}

`admission.datakit/ddtrace.enabled: "false"` を指定すると、Pod 単位で DDTrace を無効にできます。このスイッチは `check_annotation` に関係なく常に有効です。

ルールで `check_annotation: true` を設定した場合、Pod には対応する言語のバージョン Annotation も必要です。

| 言語 | バージョン Annotation |
| --- | --- |
| Java | `admission.datakit/java-lib.version` |
| Python | `admission.datakit/python-lib.version` |
| PHP | `admission.datakit/php-lib.version` |
| Node.js | `admission.datakit/nodejs-lib.version` |

バージョン値で置き換わるのは、ルール内の `image` の tag だけです。イメージのレジストリや名前は変更されないため、同じイメージのバージョン切り替えにだけ使用できます。

複数の DDTrace ルールが一致する場合、Operator は設定順に、annotation 条件を満たす最初のルールを使用します。DDTrace と OpenTelemetry の両方に一致する場合は DDTrace が優先され、DDTrace の選択後に注入が失敗しても OpenTelemetry へはフォールバックしません。同じ Pod で2種類の自動計装を同時に有効にしないでください。

## Deployment の例 {#anno-demo}

次の Deployment は、前述の Java ルールに一致します。

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: java-demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: java-demo
  template:
    metadata:
      labels:
        app: java-demo
        admission.datakit/ddtrace-language: java
      annotations:
        admission.datakit/ddtrace.enabled: "true"
    spec:
      containers:
        - name: app
          image: example/java-demo:1.0.0
```

DDTrace を明示的に無効にするには、Annotation を次のように変更します。

```yaml
admission.datakit/ddtrace.enabled: "false"
```

## 検証とトラブルシューティング {#ddtrace-verify}

Pod を再作成した後、最終的な Pod を確認します。

```shell
kubectl get pod <pod-name> -o jsonpath='{.spec.initContainers[*].name}'
kubectl get pod <pod-name> -o yaml
kubectl logs -n datakit deployment/datakit-operator
```

`datakit-lib-init`、`datakit-auto-instrument` ボリューム、対応する言語のマウント、および起動環境が含まれている必要があります。その後、実際のリクエストを送信し、DataKit monitor または <<<custom_key.brand_name>>> の画面で Trace データを確認してください。

主な問題：

- 実行中の Pod が変わらない：Pod を再作成してください。Operator が処理するのは `CREATE` だけです。
- 注入されない：Namespace、Label、および `check_annotation` の条件をすべて満たしていることを確認してください。
- init Container の pull に失敗する：ルール内のイメージアドレス、認証情報、およびクラスターネットワークを確認してください。
- init Container は成功するが Trace がない：アプリケーションプロセスが注入された起動環境を維持していることと、`DD_AGENT_HOST` および `DD_TRACE_AGENT_PORT` へ接続できることを確認してください。
- PHP の起動に失敗する：アプリケーションイメージが glibc と musl のどちらを使用しているかを確認し、正しい `php_loader_flavor` を設定してください。
