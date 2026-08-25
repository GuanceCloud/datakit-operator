# DataKit Operator による DDTrace の注入

DataKit Operator は Pod の**作成時**に admission webhook を介して Pod テンプレートを変更します。`datakit-lib-init` initContainer を追加し、言語ライブラリを共有ボリューム `/datadog-lib` にコピーしてから、そのボリュームと起動に必要な環境変数をアプリケーションコンテナに注入します。すでに実行中の Pod は変更されません。Operator ConfigMap、イメージバージョン、または Deployment のアノテーションを変更した場合、ロールアウトまたは再起動によって新しい Pod を作成すると変更が反映されます。

使用する前に、次の 3 点を確認してください。

1. セレクターによって注入対象のワークロードが決まります。空のセレクターは影響範囲を拡大する可能性があるため、まず専用の namespace で試行してください。
1. `image` のライブラリバージョンは、initContainer のランタイムではなく、**アプリケーションコンテナ**の言語ランタイムバージョンと一致させる必要があります。
1. ライブラリを注入しても、対象アプリケーションが必ず起動したり、トレースを生成したりするとは限りません。アプリケーションコンテナの起動引数、環境変数、実際の trace リクエストを引き続き確認する必要があります。

## 使用方法 {#datakit-operator-inject-lib-usage}

1. 対象の Kubernetes クラスターで、[DataKit-Operator をダウンロードしてインストール](datakit-operator.md#install)します。
1. Operator に次の ConfigMap 設定を追加します。

    ```json
    {
        "server_listen": "0.0.0.0:9543",
        "log_level": "info",
        "admission_inject_v2": {
            "ddtraces": [
                {
                    "namespace_selectors": ["staging"],
                    "label_selectors": ["app=example"],
                    "check_annotation": false,
                    "image": "<ddtrace-library-image>",
                    "language": "java",
                    "envs": {
                        "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                        "DD_TRACE_AGENT_PORT": "9529"
                    }
                }
            ]
        },
        "admission_inject": {
            "ddtrace": {}
        }
    }
    ```

    上記の例は解析可能な JSON です。`admission_inject_v2`（Operator `v1.8.0+`）は複数の DDTrace 設定をサポートしているため、優先的に使用することを推奨します。`admission_inject` は旧バージョンとの互換性を維持するための設定で、通常は 1 セットの DDTrace ルールしか指定できません。`//` コメントを含む JSON をそのまま ConfigMap にコピーしないでください。

    DDTrace 注入では、次のフィールドを設定できます。

    | フィールド                    | 型      | 説明                                                     | 必須          | 例                               |
    | ------:                      | :-----: | :------                                                  | :---:        | :--------                        |
    | `envs`                       | object  | 環境変数のマッピング                                     | Y[^envs]     | 以下の例を参照                   |
    | `image`                      | string  | DDTrace イメージのアドレス                                | Y[^image]    | 以下の例を参照                   |
    | `label_selectors`            | array   | ラベルセレクターの配列                                   | Y[^selector] | `["app=nginx", "tier=frontend"]` |
    | `language`                   | string  | サポート対象の言語タイプ（`java`/`python`/`php`/`nodejs` から選択） | Y[^lang]     | `"nodejs"`                       |
    | `namespace_selectors`        | array   | 正規表現を使用する namespace セレクター                   | Y[^selector] | `["^prod-.*$", "^test$"]`       |
    | `resources`                  | object  | リソース制限設定                                         | N            | 以下の例を参照                   |
    | ~~`enabled_namespaces`~~     | object  | 注入対象の Kubernetes namespace と対応する開発言語を指定 | Y            | `admission_inject_v2` は 1.7.0 で非推奨|
    | ~~`enabled_labelselectors`~~ | object  | Kubernetes label を使用して注入対象を選択                | Y            | `admission_inject_v2` は 1.7.0 で非推奨|

    [^selector]: フィールド自体への指定は必須です。指定しない場合、Operator は注入を拒否します。空の配列を指定すると選択範囲が広がるため、本番導入前に想定どおりの範囲であることを確認してください。
    [^image]: インストールテンプレートにはデフォルトのイメージアドレスが用意されています。オフライン環境では通常、イメージを内部ネットワークにコピーし、内部ネットワークのイメージアドレスを使用する必要があります。
    [^lang]: ここで選択する言語は、対応する DDTrace イメージの内容と一致させる必要があります。一致しない場合、注入に失敗します。
    [^envs]: これらの環境変数設定は非常に重要であり、最終的なデータに直接影響します。ここでサポートされる `fieldRef` の一覧については、[こちら](datakit-operator.md#downwardapi)を参照してください。

    `language` フィールドで許可されている値は、すべてのバージョンに利用可能なイメージが存在することを意味するものではありません。このページでは、検証済みの Node.js と Python のバージョン対応を示します。Java では以下の例を使用し、最終的な JVM コマンドで Agent がロードされていることを確認してください。PHP では、インストールテンプレートまたはリリースノートで PHP の実行方式との対応が明記されたイメージのみを使用してください。ある言語向けのイメージを別の言語ランタイムに注入しないでください。

### まず小規模で試行 {#pilot}

まずテスト用 namespace と明確な `app=<name>` ラベルセレクターを使用し、その後、段階的に対象範囲を広げることを推奨します。イメージタグは具体的なバージョンに固定し、本番ルールでは `latest` を使用しないでください。変更するたびに、少なくとも次の内容を確認してください。

```shell
kubectl get pod <pod-name> -o jsonpath='{.spec.initContainers[*].name}'
kubectl describe pod <pod-name>
```

最初のコマンドの出力には `datakit-lib-init` が含まれている必要があります。2 番目のコマンドでは、マウント、環境変数、webhook イベントを確認します。その後、アプリケーションコンテナで言語の起動引数を確認し、DataKit monitor で trace リクエストを確認する必要があります。

### DDTrace Lib の注入方法とイメージの選択 {#ddtrace-lib-image-selection}

DataKit Operator が DDTrace を注入するとき、`datakit-lib-init` という名前の initContainer を追加し、DDTrace ライブラリを共有ボリューム `/datadog-lib` にコピーしてから、そのディレクトリをアプリケーションコンテナにマウントします。イメージバージョンは、initContainer のランタイムではなく、**アプリケーションコンテナ内の言語ランタイムバージョン**に基づいて選択する必要があります。

イメージリポジトリはブランドごとに異なります。本文の例では `pubrepo.<<<custom_key.brand_main_domain>>>/datakit-operator` を統一して使用し、各ブランドのサイトに公開する際に対応するリポジトリアドレスへ置き換えます。

#### Node.js の注入 {#ddtrace-nodejs-injection}

Node.js の注入では、アプリケーションコンテナに `NODE_OPTIONS` を設定または追記します。

```shell
--require=/datadog-lib/node_modules/dd-trace/init
```

アプリケーションコンテナに `NODE_OPTIONS` がすでに存在する場合、Operator は元の値の後に上記の引数を追記します。Node.js イメージは、アプリケーションコンテナ内の Node.js メジャーバージョンと一致させる必要があります。

アプリケーションが `NODE_OPTIONS` を上書きすると、注入された引数が失われ、trace が生成されなくなります。デプロイ後は `kubectl exec` で `NODE_OPTIONS` を確認し、その中に `--require=/datadog-lib/node_modules/dd-trace/init` が保持されていることを確認できます。

| アプリケーションコンテナの Node.js バージョン | 推奨イメージ | バージョン要件 |
| --- | --- | --- |
| Node.js 18～25 | `{{.DDTraceNodeJSImage}}` | デフォルトの推奨バージョンです。`dd-trace@5.102.0` を内蔵し、`node >=18 <26` が必要です |
| Node.js 16 | `{{.DDTraceNodeJS16Image}}` | `dd-trace` 4.x シリーズを使用します。`5.102.0` を Node.js 16 で使用することは推奨しません |
| Node.js 14 以下 | デフォルトのサポート対象外 | より古い `dd-trace` メジャーバージョンと対応するイメージが必要です。Node.js のアップグレードを優先することを推奨します |

Node.js DDTrace の設定例：

```json
{
    "namespace_selectors": ["default"],
    "label_selectors": [],
    "check_annotation": false,
    "image": "{{.DDTraceNodeJSImage}}",
    "language": "nodejs",
    "envs": {
        "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
        "DD_TRACE_AGENT_PORT": "9529",
        "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
        "POD_NAME": "{fieldRef:metadata.name}",
        "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
        "NODE_NAME": "{fieldRef:spec.nodeName}",
        "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
    }
}
```

#### Python の注入 {#ddtrace-python-injection}

Python の注入では、アプリケーションコンテナに `PYTHONPATH=/datadog-lib/` を設定または追記し、Python プロセスが `/datadog-lib` から DDTrace 関連ライブラリと注入用 bootstrap をロードできるようにします。Python の DDTrace には CPython ABI に関連する wheel が含まれるため、イメージバージョンはアプリケーションコンテナの Python マイナーバージョンと一致させる必要があります。バージョンが一致しない場合、`ModuleNotFoundError`、native extension のロード失敗、または起動失敗が発生する可能性があります。

アプリケーションイメージまたは起動スクリプトで `PYTHONPATH` を上書きする場合は、`/datadog-lib/` を保持する必要があります。保持しない場合、注入されたライブラリをロードできません。起動後に `python -c 'import ddtrace; print(ddtrace.__version__)'` を実行し、最小限のロード確認を行えます。

| アプリケーションコンテナの Python バージョン | 推奨イメージ | バージョン要件 |
| --- | --- | --- |
| Python 3.7 | `{{.DDTracePython37Image}}` | `ddtrace` 2.x は Python 3.7 をサポートします。Python 3.7 は `ddtrace` 3.x/4.x をサポートしません |
| Python 3.8 | `{{.DDTracePython38Image}}` | `ddtrace` 3.x は Python 3.8 をサポートします。`ddtrace` 4.x には Python 3.9 以降が必要です |
| Python 3.9～3.14 | `{{.DDTracePythonImage}}` | `ddtrace` 4.x では現在 `python >=3.9,<3.15` が必要です |

Python DDTrace の設定例：

```json
{
    "namespace_selectors": ["default"],
    "label_selectors": [],
    "check_annotation": false,
    "image": "{{.DDTracePythonImage}}",
    "language": "python",
    "envs": {
        "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
        "DD_TRACE_AGENT_PORT": "9529",
        "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
        "POD_NAME": "{fieldRef:metadata.name}",
        "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
        "NODE_NAME": "{fieldRef:spec.nodeName}",
        "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
    }
}
```

> バージョン選択は、DDTrace アップストリームパッケージのランタイム要件に基づきます。Node.js は npm `dd-trace` の `engines.node`、Python は PyPI `ddtrace` の `Requires-Python` と wheel のサポートを基準とします。

#### Java の注入確認 {#ddtrace-java-injection}

Java の注入では、ライブラリボリュームに加えて、JVM に `-javaagent` を実際にロードさせる必要があります。アプリケーションの起動後、最終的な Pod の `JAVA_TOOL_OPTIONS`、コンテナの command/args、またはプロセスのコマンドラインを確認し、Operator が注入した Agent のパスが含まれていることを確認してください。その後、`DD_AGENT_HOST` と `DD_TRACE_AGENT_PORT=9529` を確認します。`datakit-lib-init` の成功だけを確認しても、JVM が Agent をロードしたことの証明にはなりません。

### `check_annotation` 設定項目の説明 {#check-annotation-config}

`check_annotation` は、DataKit Operator が Pod の**バージョンアノテーション**（`admission.datakit/java-lib.version`、`admission.datakit/python-lib.version`、`admission.datakit/nodejs-lib.version` など）をどのように処理するかを制御する重要な設定フィールドです。このフィールドの値と動作は次のとおりです。

| 値       | 動作                                                                     |
|----------|--------------------------------------------------------------------------|
| `false`  | **（デフォルト値）** Pod の**バージョンアノテーション**の確認を無視し、セレクタールールに基づいて直接注入します |
| `true`   | **バージョンアノテーション**の確認を有効にし、一致する Pod にバージョンアノテーションが存在する場合のみ注入します |

#### 重要なロジック {#important-logic}

1. **機能固有のアノテーションは常に有効です**：
   - `admission.datakit/ddtrace.enabled` は `check_annotation` 設定の影響を**受けません**
   - `check_annotation` が `true` と `false` のどちらであっても、`admission.datakit/ddtrace.enabled` が確認されます
   - `admission.datakit/ddtrace.enabled: "false"` の場合、注入は直ちに拒否されます

2. **バージョンアノテーションは `check_annotation` によって制御されます**：
   - `admission.datakit/<language>-lib.version` は `check_annotation` 設定の影響を**受けます**
   - `check_annotation: true` の場合、バージョンアノテーションが存在するときのみ注入されます
   - `check_annotation: false` の場合、バージョンアノテーションの確認は無視されます

3. **グローバルアノテーションは常に有効です**：
   - `admission.datakit/enabled` は `check_annotation` 設定の影響を**受けません**
   - `admission.datakit/enabled: "false"` の場合、すべての注入が完全に拒否されます（最優先）

サポートされる DDTrace 関連の Annotation：

| Annotation                           | 機能                                   | 値               | `check_annotation` の影響 | 説明                                                                 |
|--------------------------------------|----------------------------------------|------------------|---------------------------|----------------------------------------------------------------------|
| `admission.datakit/ddtrace.enabled`  | DDTrace 注入を制御                     | `"true"`/`"false"` | **いいえ**               | `"true"`：注入を許可、`"false"`：注入を拒否、未設定：ルールの一致結果に基づいて決定 |
| `admission.datakit/java-lib.version` | DDTrace Java Agent のバージョンを指定  | バージョン文字列 | **はい**                 | `"1.12.0"` など、設定内のデフォルトイメージバージョンを上書きするために使用します |
| `admission.datakit/python-lib.version` | DDTrace Python Lib のバージョンを指定  | バージョン文字列 | **はい**                 | `"v3.19.7"` など、設定内のデフォルトイメージバージョンを上書きするために使用します |
| `admission.datakit/nodejs-lib.version` | DDTrace Node.js Lib のバージョンを指定 | バージョン文字列 | **はい**                 | `"5.102.0"` など、設定内のデフォルトイメージバージョンを上書きするために使用します |
| `admission.datakit/enabled`          | すべての注入機能を制御（最優先）       | `"true"`/`"false"` | **いいえ**               | `"false"`：すべての注入を完全に拒否し、最も高い優先度を持ちます |

#### `check_annotation: true` の場合 {#when-check-annotation-true}

次の条件をすべて満たした場合にのみ注入されます。

1. **設定の一致**：Pod が `namespace_selectors` と `label_selectors` のルールに一致する必要があります
2. **機能アノテーションによる許可**：`admission.datakit/ddtrace.enabled` が `"false"` ではないこと（存在する場合）
3. **バージョンアノテーションの存在**：Pod にバージョンアノテーション（`admission.datakit/java-lib.version`、`admission.datakit/python-lib.version`、`admission.datakit/nodejs-lib.version` など）が存在する必要があります

#### `check_annotation: false` の場合 {#when-check-annotation-false}

次の条件を満たした場合に注入されます。

1. **設定の一致**：Pod が `namespace_selectors` と `label_selectors` のルールに一致する必要があります
2. **機能アノテーションによる許可**：`admission.datakit/ddtrace.enabled` が `"false"` ではないこと（存在する場合）
3. **バージョンアノテーションを無視**：バージョンアノテーションがなくても注入されます

#### ユースケースの例 {#use-case-examples}

1. **厳密なバージョン管理のユースケース**（`check_annotation: true`）：

   ```json
   {
       "namespace_selectors": ["prod"],
       "label_selectors": ["app=backend"],
       "check_annotation": true,
       "image": "internal-registry/dd-lib-java:<pinned-version>",
       "language": "java"
   }
   ```

   **注入条件**：
   - Pod が `prod` namespace にあり、`app=backend` ラベルが付いていること
   - Pod に `admission.datakit/ddtrace.enabled: "false"` が**ない**こと（存在する場合）
   - Pod に `admission.datakit/java-lib.version` アノテーションが**必ず存在する**こと

2. **一括注入のユースケース**（`check_annotation: false`）：

   ```json
   {
       "namespace_selectors": ["staging"],
       "label_selectors": ["env=test"],
       "check_annotation": false,
       "image": "internal-registry/dd-lib-java:<pinned-version>",
       "language": "java"
   }
   ```

   **注入条件**：
   - Pod が `staging` namespace にあり、`env=test` ラベルが付いていること
   - Pod に `admission.datakit/ddtrace.enabled: "false"` が**ない**こと（存在する場合）
   - `admission.datakit/java-lib.version` アノテーションの確認を**無視**します

3. **選択的に拒否するユースケース**：

   ```json
   {
       "namespace_selectors": ["prod"],
       "label_selectors": ["app=java-app"],
       "check_annotation": false,
       "image": "internal-registry/dd-lib-java:<pinned-version>",
       "language": "java"
   }
   ```

   **注入ロジック**：
   - 一致するすべての Pod に注入されます
   - Pod に `admission.datakit/ddtrace.enabled: "false"` がある場合、その Pod は除外されます
   - バージョンアノテーション `admission.datakit/java-lib.version: "1.15.0"` を使用してイメージバージョンを上書きできますが、注入の実行可否には影響しません

    以下に例を示します。

    ```json
    {
        "namespace_selectors": [],
        "check_annotation": false,
        "label_selectors": [],
        "image": "{{.DDTraceJavaImage}}",
        "language": "java",
        "envs": {
             "DD_AGENT_HOST":           "datakit-service.datakit.svc.cluster.local",
             "DD_TRACE_AGENT_PORT":     "9529",
             "DD_JMXFETCH_STATSD_HOST": "datakit-service.datakit.svc.cluster.local",
             "DD_JMXFETCH_STATSD_PORT": "8125",
             "DD_SERVICE":              "{fieldRef:metadata.labels['service']}",
             "POD_NAME":                "{fieldRef:metadata.name}",
             "POD_NAMESPACE":           "{fieldRef:metadata.namespace}",
             "NODE_NAME":               "{fieldRef:spec.nodeName}",
             "DD_TAGS":                 "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
         },
        "resources": {
            "requests": {
                "cpu":    "100m",
                "memory": "64Mi"
            },
            "limits": {
                "cpu":    "500m",
                "memory": "512Mi"
            }
        }
    }
    ```

## 特定の Deployment の処理 {#special-deployment}

上記の Operator 設定は、クラスター全体の DDTrace 注入に適用されます。ただし、この一律の方法が特定の Deployment に適さない場合があります。そのため、これらの Deployment に個別の Annotation を付与できます。

Operator は次の Annotation を識別できます。

- `admission.datakit/ddtrace.enabled`：個別の Deployment で注入を有効にするかどうかを指定します。`"true"` を指定すると注入が有効になり、`"false"` を指定すると注入が無効になります。無効にすると、Operator はこの Deployment への注入をスキップします
- `admission.datakit/java-lib.version`：特定の DDTrace Java Agent バージョンを指定します
- `admission.datakit/python-lib.version`：特定の DDTrace Python Lib バージョンを指定します
- `admission.datakit/nodejs-lib.version`：特定の DDTrace Node.js Lib バージョンを指定します

> **アノテーションの使用方法**：`check_annotation` 設定がバージョンアノテーションの動作に与える影響については、[Annotation による注入設定](datakit-operator.md#annotation-injection)と[本ページの `check_annotation` 設定項目の説明](operator-ddtrace.md#check-annotation-config)を参照してください。

### Annotation の例 {#anno-demo}

Deployment に注入の有効または無効を示すマーカーを付与します。

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app-deployment
  labels:
    app: my-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
      annotations:
        admission.datakit/ddtrace.enabled: "true"
    spec:
      containers:
      - name: my-app
        image: my-app:1.2.3
        ports:
        - containerPort: 80
```

Deployment に `dd-java-lib` の特定のバージョン番号を注入します [^replace-ddtrace-version]。

[^replace-ddtrace-version]: 此处替换的是 Operator ConfigMap 内の同じイメージアドレスの別バージョンを指定します。ここで別のイメージアドレスに切り替えることはできません。

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app-deployment
  labels:
    app: my-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
      annotations:
        admission.datakit/java-lib.version: "<version>"
    spec:
      containers:
      - name: my-app
        image: my-app:1.2.3
        ports:
        - containerPort: 80
```

YAML ファイルを使用してリソースを作成します。

```shell
$ kubectl apply -f my-app.yaml
...
```

次のように確認します。

```shell
$ kubectl get pod

NAME                                   READY   STATUS    RESTARTS      AGE
my-app-deployment-7bd8dd85f-fzmt2       1/1     Running   0             4s

$ kubectl get pod my-app-deployment-7bd8dd85f-fzmt2 -o=jsonpath={.spec.initContainers\[\*\].name}

datakit-lib-init
```
