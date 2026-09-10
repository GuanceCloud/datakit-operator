# DataKit Operator による OpenTelemetry の注入

DataKit Operator は [:octicons-tag-24: v1.9.0](operator-changelog.md#cl-1.9.0) 以降、Java、Python、および Node.js アプリケーションへの OpenTelemetry 自動計装の注入をサポートします。この機能は OpenTelemetry 公式の自動計装イメージを使用し、そのプローブのコピー方法と起動環境の設定方法に従います。

配布テンプレートでは `otels` はデフォルトで空のため、OpenTelemetry の注入は自動的に有効になりません。有効にするには、まずこのページに示すルールを追加してください。注入は Pod の作成時にだけ行われます。Operator は Pod 内のすべての通常のアプリケーションコンテナを変更しますが、アプリケーションの init Container は変更せず、コンテナ内の言語バージョンや libc も検出しません。

## 使用前の準備 {#otel-prerequisites}

### DataKit OpenTelemetry コレクターの有効化 {#enable-datakit-otel}

DataKit で `opentelemetry` input を有効にする必要があります。たとえば、DataKit DaemonSet のデフォルトコレクター一覧に `opentelemetry` を追加します。

```yaml
- name: ENV_DEFAULT_ENABLED_INPUTS
  value: statsd,dk,cpu,ddtrace,opentelemetry
```

OTLP Trace、Metric、および Log は同じ DataKit Service と `9529` ポートを使用しますが、リクエストパスが異なります。

```text
Trace:  http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/traces
Metric: http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/metrics
Log:    http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/logs
```

このページのサンプルルールでは Trace だけが有効です。Metric または Log を収集するには、対応する `OTEL_METRICS_EXPORTER` または `OTEL_LOGS_EXPORTER` を `none` から `otlp` に変更します。新しい DataKit アドレスを設定する必要はありません。アプリケーションと自動計装自体も、対応するシグナルを生成できる必要があります。

### 対応言語の範囲 {#otel-language-support}

| 言語 | サポート範囲 | デフォルトイメージ |
| --- | --- | --- |
| Java | OpenTelemetry Java Agent が対応する JVM | `{{.OTelJavaImage}}` |
| Python | Python 3.10～3.14。glibc Linux のみ | `{{.OTelPythonImage}}` |
| Node.js | Node.js 20.6 以降。glibc と Alpine/musl をサポート | `{{.OTelNodeJSImage}}` |

Python 2、Python 3.9 以前、および Alpine/musl の Python イメージはサポート対象外です。Node.js は現在、通常の CommonJS アプリケーションをサポートしますが、ESM、bundler、カスタム loader はサポートしません。OpenTelemetry 公式 Operator は現時点で PHP の自動注入を正式にはサポートしていません。公式の PHP 自動計装イメージは提供されていますが、対応する Instrumentation 設定と標準的な注入フローはまだ提供されていないため、DataKit Operator でも OTel PHP の注入はサポートしていません。

Operator はこれらの条件を自動判定しません。相互排他的な Namespace または Label Selector を使用し、条件を満たす Pod だけが対応するルールに一致するようにしてください。

## Operator の設定 {#otel-config}

`otels` は `ddtraces` と同じ階層にあり、配布テンプレートでのデフォルト値は `[]` です。次の設定には Java、Python、および Node.js のルールがすべて含まれています。既存の `jsonconfig` を編集する場合は、この `otels` 配列全体で `admission_inject_v2.otels` を置き換え、`admission_inject_v2` 配下のその他の設定は維持してください。

すべてのルールで `"namespace_selectors": ["*"]` を使用しますが、対応する言語ラベルを持つ Pod だけが一致します。そのため、ラベルのない Pod に自動注入することなく、Namespace をまたいで利用できます。3 つのルールはデフォルトで Trace だけをエクスポートします。

```json
{
    "admission_inject_v2": {
        "otels": [
            {
                "name": "otel-java",
                "language": "java",
                "namespace_selectors": ["*"],
                "label_selectors": ["admission.datakit/otel-language=java"],
                "check_annotation": false,
                "image": "{{.OTelJavaImage}}",
                "envs": {
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "OTEL_SERVICE_NAME": "{fieldRef:metadata.labels['app']}",
                    "OTEL_RESOURCE_ATTRIBUTES": "k8s.pod.name=$(POD_NAME),k8s.namespace.name=$(POD_NAMESPACE),k8s.node.name=$(NODE_NAME)",
                    "OTEL_TRACES_EXPORTER": "otlp",
                    "OTEL_LOGS_EXPORTER": "none",
                    "OTEL_METRICS_EXPORTER": "none",
                    "OTEL_EXPORTER_OTLP_PROTOCOL": "http/protobuf",
                    "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/traces",
                    "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/logs",
                    "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/metrics"
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
                "name": "otel-python",
                "language": "python",
                "namespace_selectors": ["*"],
                "label_selectors": ["admission.datakit/otel-language=python"],
                "check_annotation": false,
                "image": "{{.OTelPythonImage}}",
                "envs": {
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "OTEL_SERVICE_NAME": "{fieldRef:metadata.labels['app']}",
                    "OTEL_RESOURCE_ATTRIBUTES": "k8s.pod.name=$(POD_NAME),k8s.namespace.name=$(POD_NAMESPACE),k8s.node.name=$(NODE_NAME)",
                    "OTEL_TRACES_EXPORTER": "otlp",
                    "OTEL_LOGS_EXPORTER": "none",
                    "OTEL_METRICS_EXPORTER": "none",
                    "OTEL_EXPORTER_OTLP_PROTOCOL": "http/protobuf",
                    "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/traces",
                    "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/logs",
                    "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/metrics"
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
                "name": "otel-nodejs",
                "language": "nodejs",
                "namespace_selectors": ["*"],
                "label_selectors": ["admission.datakit/otel-language=nodejs"],
                "check_annotation": false,
                "image": "{{.OTelNodeJSImage}}",
                "envs": {
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "OTEL_SERVICE_NAME": "{fieldRef:metadata.labels['app']}",
                    "OTEL_RESOURCE_ATTRIBUTES": "k8s.pod.name=$(POD_NAME),k8s.namespace.name=$(POD_NAMESPACE),k8s.node.name=$(NODE_NAME)",
                    "OTEL_TRACES_EXPORTER": "otlp",
                    "OTEL_LOGS_EXPORTER": "none",
                    "OTEL_METRICS_EXPORTER": "none",
                    "OTEL_EXPORTER_OTLP_PROTOCOL": "http/protobuf",
                    "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/traces",
                    "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/logs",
                    "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/metrics"
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

環境変数は設定順を維持します。すべてのルールで、`POD_NAME`、`POD_NAMESPACE`、および `NODE_NAME` を、それらを参照する `OTEL_RESOURCE_ATTRIBUTES` より前に配置する必要があります。

主なフィールドは次のとおりです。

| フィールド | 説明 |
| --- | --- |
| `name` | ルール名。ログでの特定に使用するため、設定を推奨します |
| `language` | 必須。`java`、`python`、または `nodejs` を指定できます |
| `namespace_selectors` | Namespace の正規表現配列 |
| `label_selectors` | Pod Label Selector の配列 |
| `check_annotation` | 対応する言語のバージョン Annotation を Pod に要求するかどうか。デフォルトは `false` |
| `image` | 必須。OpenTelemetry 公式イメージ、またはプライベートレジストリに複製したイメージ |
| `image_pull_policy` | 任意。`Always`、`IfNotPresent`、`Never`。未指定または不正な値は `Always` を使用します。[イメージの pull ポリシー](datakit-operator.md#image-pull-policy)を参照してください |
| `envs` | すべての通常のアプリケーションコンテナに注入する環境変数 |
| `resources` | init Container のリソース設定。未指定または無効な場合はデフォルト値を使用します |

3 つのルールで使用する言語ラベルとイメージは次のとおりです。

| 言語 | Label | イメージ |
| --- | --- | --- |
| Java | `admission.datakit/otel-language=java` | `{{.OTelJavaImage}}` |
| Python | `admission.datakit/otel-language=python` | `{{.OTelPythonImage}}` |
| Node.js | `admission.datakit/otel-language=nodejs` | `{{.OTelNodeJSImage}}` |

Pod には対応する言語ラベルを 1 つだけ設定します。配布テンプレートのデフォルト Java DDTrace ルールは `default` Namespace の Pod に一致するため、OTel Pod では `admission.datakit/ddtrace.enabled: "false"` も明示的に設定してください。完全な Deployment は後述の例を参照してください。

Selector と Annotation の共通ルールについては、[DataKit Operator の注入ルール](datakit-operator.md#datakit-operator-inject)を参照してください。同じ Pod が複数の OTel ルールに一致する場合、Operator は annotation 条件を満たす最初のルールだけを使用します。最初のルールの言語またはイメージ設定が正しくない場合も、後続ルールへはフォールバックしません。

## 言語別の注入方式 {#otel-injection}

すべての言語で、次の内容が追加されます。

- `datakit-otel-lib-init` init Container。
- `datakit-otel-auto-instrument` EmptyDir ボリューム。
- ルールで設定した `OTEL_*` などの環境変数。

言語ごとのプローブの読み込み方法は次のとおりです。

| 言語 | init Container でのコピー方法 | マウントと起動環境 |
| --- | --- | --- |
| Java | イメージ内の `/javaagent.jar` を共有ボリュームへコピー | `/otel-auto-instrumentation-java` をマウントし、`JAVA_TOOL_OPTIONS` に `-javaagent:/otel-auto-instrumentation-java/javaagent.jar` を追加 |
| Python | イメージ内の `/autoinstrumentation/` を共有ボリュームへコピー | `/otel-auto-instrumentation-python` をマウントし、`PYTHONPATH` の先頭と末尾に自動初期化ディレクトリとプローブディレクトリを追加して、`sitecustomize.py` から読み込み |
| Node.js | イメージ内の `/autoinstrumentation/` を共有ボリュームへコピー | `/otel-auto-instrumentation-nodejs` をマウントし、`NODE_OPTIONS` に `--require /otel-auto-instrumentation-nodejs/autoinstrumentation.js` を追加 |

Operator は、既存の通常の文字列値を維持します。対応する起動環境が `valueFrom` を使用している場合、重複項目がある場合、Datadog プローブがすでに読み込まれている場合、または既存の OTel init Container、ボリューム、マウントが想定と競合する場合、Operator は OTel の注入全体をスキップして warning を記録します。Admission は引き続き fail-open で動作し、アプリケーション Pod の作成を妨げません。

## Annotation とバージョン {#otel-annotations}

`admission.datakit/otel.enabled: "false"` を指定すると、Pod 単位で OTel を無効にできます。このスイッチは `check_annotation` に関係なく常に有効です。

ルールで `check_annotation: true` を設定した場合、Pod には対応する言語のバージョン Annotation も必要です。

| 言語 | バージョン Annotation |
| --- | --- |
| Java | `admission.datakit/otel-java-lib.version` |
| Python | `admission.datakit/otel-python-lib.version` |
| Node.js | `admission.datakit/otel-nodejs-lib.version` |

バージョン値で置き換わるのは、ルール内の `image` の tag だけです。イメージのレジストリや名前は変更されません。プラットフォーム側でバージョンを一元管理する場合は、`check_annotation: false` のままにすることを推奨します。

## DDTrace との関係 {#otel-ddtrace-conflict}

同じコンテナで DDTrace と OpenTelemetry の自動計装を同時に読み込まないでください。DDTrace と OTel の両方のルールに一致する場合は DDTrace が優先され、DDTrace の選択後に注入が失敗しても OTel へはフォールバックしません。

OTel ワークロードでは、DDTrace を明示的に無効にすることを推奨します。

```yaml
admission.datakit/ddtrace.enabled: "false"
```

## `runAsNonRoot` の保護 {#otel-run-as-non-root}

OpenTelemetry 公式の自動計装イメージで使用されるデフォルトユーザーは、アプリケーション Pod のセキュリティポリシーと一致しない場合があります。Operator は最初のアプリケーションコンテナの SecurityContext を引き継いで OTel init Container を作成します。

init Container の実効設定が `runAsNonRoot: true` で、ゼロ以外の `runAsUser` が明示されていない場合、イメージが非 root ユーザーで実行されることを確認できず、Kubernetes が起動を拒否する可能性があります。注入によってアプリケーション Pod が初期化段階で停止することを防ぐため、Operator は OTel の注入全体をスキップし、次の理由を含む warning を記録します。

```text
reason=run_as_non_root_without_run_as_user
```

このような Pod で OTel を有効にするには、最初のアプリケーションコンテナまたは Pod に、使用する OTel イメージと互換性のあるゼロ以外の `runAsUser` を明示してください。`runAsNonRoot: true` の場合、プライベートイメージが実際には非 root ユーザーを使用していても、`runAsUser` が明示されていなければ安全のためスキップされます。

## Deployment の例 {#otel-example}

次の Java Deployment は、前述の設定に一致します。

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: order-service
  namespace: production
spec:
  replicas: 1
  selector:
    matchLabels:
      app: order-service
  template:
    metadata:
      labels:
        app: order-service
        admission.datakit/otel-language: java
      annotations:
        admission.datakit/ddtrace.enabled: "false"
        admission.datakit/otel.enabled: "true"
    spec:
      containers:
        - name: app
          image: example/order-service:1.0.0
          ports:
            - name: http
              containerPort: 8080
```

ラベルの値を `python` または `nodejs` に変更すると、対応する言語ルールに一致します。アプリケーションイメージは、前述のサポート範囲を満たす必要があります。

## 検証とトラブルシューティング {#otel-verify}

Pod の作成後、注入結果を確認します。

```shell
kubectl -n production get pod -l app=order-service -o yaml
kubectl logs -n datakit deployment/datakit-operator
```

最終的な Pod には、`datakit-otel-lib-init`、`datakit-otel-auto-instrument`、対応する言語のマウント、起動環境、および `OTEL_*` 環境変数が含まれている必要があります。その後、アプリケーションへ実際のリクエストを送信し、画面で次の条件を使用して検索します。

```text
service:order-service
source:opentelemetry
```

主な問題：

- 実行中の Pod が変わらない：Pod を再作成してください。Operator が処理するのは `CREATE` だけです。
- 注入されない：Namespace、Label、`check_annotation`、DDTrace の優先順位、および Operator の warning を確認してください。
- init Container の pull に失敗する：GHCR へのネットワーク接続を確認するか、イメージをプライベートレジストリへ同期してルールを変更してください。デフォルトの pull ポリシーは `Always` で、ルールの `image_pull_policy` で変更できます。`IfNotPresent` または `Never` を使う場合は、ノード上のイメージの有無を確認してください。
- 注入されているがデータがない：DataKit で `opentelemetry` input が有効であること、OTLP アドレスへ接続できること、アプリケーションフレームワークが自動計装に対応していることを確認し、実際のリクエストを送信してください。
- ロールバックする：OTel ルールを削除または無効にしてから、注入済みの Pod を再作成してください。
