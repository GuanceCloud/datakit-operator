# DataKit Operator

---

:material-kubernetes:

---

DataKit Operator は、Kubernetes オーケストレーションと DataKit を連携するプロジェクトです。DataKit のデプロイを容易にし、検証や注入などの機能を提供することを目的としています。

## 概要 {#overview}

DataKit Operator は、Kubernetes Admission Controller の仕組みを通じて Kubernetes クラスターに自動注入機能を提供し、オブザーバビリティ機能を容易に統合できるようにします。主な機能は次のとおりです。

- **DDTrace 注入**：Java アプリケーションに APM トレーシングエージェントを自動注入します
- **ログ収集**：logfwd Sidecar を通じてコンテナログを自動収集します
- **パフォーマンス分析**：Flameshot または Profiler コンポーネントを注入し、アプリケーションパフォーマンスを監視します
- **設定管理**：グローバル設定と宣言型設定による 2 種類の注入方式をサポートします
- **Cluster API**：クラスター内の Pod を照会するプロキシを提供し、DataKit などのコンポーネントが Kubernetes メタデータを取得できるようにします

**主なメリット**：

- **自動デプロイ**：アプリケーションの YAML を手動で変更する必要がなく、設定ミスを削減します
- **一括管理**：namespace とラベルセレクターを使用して一括注入を実現します
- **柔軟な設定**：JSON 設定と Annotation による詳細な制御をサポートします
- **バージョン互換性**：後方互換性を維持し、スムーズなアップグレードをサポートします

## 前提条件 {#prerequisites}

- Kubernetes v1.24.1 以降を推奨します。また、インターネットにアクセスできる必要があります（YAML ファイルのダウンロードと対応するイメージの取得のため）
- `MutatingAdmissionWebhook` および `ValidatingAdmissionWebhook` [コントローラー](https://kubernetes.io/zh-cn/docs/reference/access-authn-authz/extensible-admission-controllers/#prerequisites){:target="_blank"} が有効になっていることを確認します
- `admissionregistration.k8s.io/v1` API が有効になっていることを確認します

## インストール {#install}

<!-- markdownlint-disable MD046 -->
=== "Deployment"

    [*datakit-operator.yaml*](https://static.<<<custom_key.brand_main_domain>>>/datakit-operator/datakit-operator.yaml){:target="_blank"} をダウンロードします。手順は次のとおりです。


    ``` shell
    $ kubectl create namespace datakit
    $ wget https://static.<<<custom_key.brand_main_domain>>>/datakit-operator/datakit-operator.yaml
    $ kubectl apply -f datakit-operator.yaml
    $ kubectl get pod -n datakit


    NAME                               READY   STATUS    RESTARTS   AGE
    datakit-operator-f948897fb-5w5nm   1/1     Running   0          15s
    ```

=== "Helm"

    前提条件

    * Kubernetes >= 1.14
    * Helm >= 3.0+

    ```shell
    $ helm install datakit-operator datakit-operator \
        <<<% if custom_key.brand_key == 'guance' -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/datakit-operator \
        <<<% else -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/truewatch \
        <<<% endif -%>>>
        -n datakit --create-namespace
    ```

    デプロイ状態を確認します。

    ```shell
    $ helm -n datakit list
    ```

    次のコマンドでアップグレードできます。

    ```shell
    $ helm -n datakit get values datakit-operator -a -o yaml > values.yaml
    $ helm upgrade datakit-operator datakit-operator \
        <<<% if custom_key.brand_key == 'guance' -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/datakit-operator \
        <<<% else -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/truewatch \
        <<<% endif -%>>>
        -n datakit \
        -f values.yaml
    ```

    次のコマンドでアンインストールできます。

    ```shell
    $ helm uninstall datakit-operator -n datakit
    ```

???+ attention

    - DataKit Operator では、プログラムと YAML が厳密に対応しています。古すぎる YAML を使用すると、新しいバージョンの DataKit Operator をインストールできない場合があります。最新の YAML を再度ダウンロードしてください。
    - `InvalidImageName` エラーが発生した場合は、イメージを手動で pull できます。
<!-- markdownlint-enable MD046 -->

### 設定 {#jsonconfig}

DataKit Operator の設定は JSON 形式です。Kubernetes では個別の ConfigMap に保存され、環境変数としてコンテナに読み込まれます。

<!-- markdownlint-disable MD046 -->
=== "DataKit Operator >= v1.8.0"

    DataKit-Operator v1.8.0 以降では、`admission_inject_v2` 設定項目の使用を推奨します。新しい設定では配列構造を採用し、より柔軟な設定方式をサポートしています。

    ```json
    {
        "server_listen": "0.0.0.0:9543", // Operator 自身のサービスリッスンアドレス
        "log_level": "info",             // Operator 自身のログレベル
        "admission_inject_v2": {         // 注入設定 v2
            "ddtraces": [...],           // DDTrace 設定配列
            "logfwds": [...],            // ログ転送設定配列
            "flameshots": [...]          // パフォーマンス分析設定配列
        },
        "admission_mutate": {            // 設定変更
            "loggings": [...]            // ログ設定の変更
        }
    }
    ```

=== "DataKit Operator < v1.8.0"

    ```json
    {
        "server_listen": "0.0.0.0:9543",
        "log_level":     "info",
        "admission_inject": {
            "ddtrace": {...},
            "profiler": {...},
            "logfwd": {...}
        },
        "admission_mutate": {
            "loggings": [...]
        }
    }
    ```
<!-- markdownlint-enable MD046 -->

## Cluster API {#cluster-api}

DataKit Operator [:octicons-tag-24: v1.8.1](operator-changelog.md#cl-1.8.1) 以降では、クラスター内の Pod データをプロキシ経由で照会する Cluster API を提供します。DataKit はこのインターフェースを通じて Kubernetes メタデータを取得できるため、各 DataKit インスタンスが API Server に直接アクセスする際の負荷を軽減できます。

Cluster API はデフォルトで有効になっており、追加の設定スイッチは不要です。Operator は起動時に、自身の ServiceAccount に Pod の読み取り権限があるかを確認します。権限がない場合、Operator はログに RBAC チェックの失敗を出力し、Cluster API 関連のルートを無効にします。最新の `datakit-operator.yaml` または Helm Chart には必要な権限が含まれています。旧バージョンの YAML からアップグレードする場合は、ClusterRole に少なくとも次の権限が含まれていることを確認してください。

```yaml
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
```

エンドポイントは Operator Service を再利用し、デフォルトのアドレスは `https://datakit-operator.datakit.svc:443` です。現在サポートされている Pod 照会インターフェースは次のとおりです。

| インターフェース | 説明 |
| --- | --- |
| `/v1/cluster/api/v1/pods` | すべての Pod の一覧を照会します |
| `/v1/cluster/api/v1/namespaces/{namespace}/pods` | 指定した namespace 内の Pod の一覧を照会します |
| `/v1/cluster/api/v1/namespaces/{namespace}/pods/{name}` | 指定した Pod を照会します |

DataKit Operator [:octicons-tag-24: v1.8.9](operator-changelog.md#cl-1.8.9) 以降では、Pod 照会インターフェースが `view=ebpf-v1` クエリパラメーターをサポートします。このビューでは、eBPF に不要な大きなフィールドを除外し、ワークロードの識別に必要な Pod の基本情報のみを保持します。これにより、大規模クラスター環境での JSON の転送および解析にかかるオーバーヘッドを削減できます。

```shell
curl -k "https://datakit-operator.datakit.svc:443/v1/cluster/api/v1/pods?view=ebpf-v1"
```

## 注入方式 {#datakit-operator-inject}

DataKit Operator は、次の 2 種類のリソース入力方式をサポートします。

1. selector 設定による注入（命令型）

    DataKit-Operator の config を変更して、対象 Pod の Namespace と Selector を指定します。条件に一致する Pod が検出されると、注入を実行します。

    **メリット**：対象 Pod に Annotation を追加する必要がありません（ただし、対象 Pod の再起動が必要です）

    **デメリット**：対象範囲の精度が十分でないため、不要な注入が発生する可能性があります

1. Annotation 設定による注入（宣言型）

    対象 Pod に Annotation を追加し、その Pod への注入を有効にします。

    **メリット**：Annotation を使用して注入を拒否するかどうかを正確に制御できます

    **デメリット**：Annotation だけで注入をトリガーすることはできず、引き続き一致ルールの設定が必要です。つまり、対象 Pod の annotation で注入を有効にするだけでなく、Operator でその他のフィールドも設定する必要があります。

### Selector 設定による注入 {#selectors-injection}

`namespace_selectors` と `label_selectors` を設定することで、一括注入を実現できます。

`admission_inject_v2` 設定では、`namespace_selectors` と `label_selectors` を配列項目内に直接設定します。DDTrace 注入の例を次に示します。

```json
{
    "admission_inject_v2": {
        "ddtraces": [
            {
                "namespace_selectors": ["testns"],
                "label_selectors":     ["app=log-output"],
                ...
            }
        ]
    }
}
```

- `namespace_selectors`：namespace セレクターの配列で、正規表現によるマッチングをサポートします。完全一致させるには、`^` と `$` でパターンを囲みます。例：`^testns$`
- `label_selectors`：Kubernetes Label Selector 構文を使用するラベルセレクターの配列です

両方の selector を設定した場合、対象 Pod は両方の条件を満たす必要があります。label selector の記述仕様については、こちらの[公式ドキュメント](https://kubernetes.io/zh-cn/docs/concepts/overview/working-with-objects/labels/#label-selectors){:target="_blank"}を参照してください。

### Annotation 設定による注入 {#annotation-injection}

Deployment に指定の Annotation を追加することで、注入を許可するかどうかを制御できます。Annotation は template 内に追加してください。

サポートされている Annotation は次のとおりです。

| Annotation                            | 機能説明            | 値             | 優先度   |
| ------------                          | ----------          | ------           | -------- |
| `admission.datakit/ddtrace.enabled`   | ddtrace 注入を制御します   | `"true"/"false"` | 中       |
| `admission.datakit/logfwd.enabled`    | logfwd 注入を制御します    | `"true"/"false"` | 中       |
| `admission.datakit/flameshot.enabled` | flameshot 注入を制御します | `"true"/"false"` | 中       |
| `admission.datakit/enabled`           | すべての注入機能を制御します    | `"true"/"false"` | **最高** |

例：

```yaml
    annotations:
    admission.datakit/ddtrace.enabled: "true"
    admission.datakit/logfwd.enabled: "true"
```

<!-- markdownlint-disable MD046 -->
???+ tip

    Annotation を使用して注入を拒否できます（`"false"` に設定します）。ただし、能動的に注入する場合は、次の設定が必要です。

    1. DataKit-Operator の設定で、一致ルール（`namespace_selectors`/`label_selectors`）と対応する設定フィールドを指定します
    1. Pod を設定内の selectors に一致させます
<!-- markdownlint-enable MD046 -->

### `check_annotation` 設定項目の説明 {#check-annotation-config}

`check_annotation` は、DataKit Operator が Pod 上の**バージョン Annotation**を処理する方法を制御する設定フィールドです。値と動作は次のとおりです。

| 値     | 動作説明                                                                 |
|----------|--------------------------------------------------------------------------|
| `false`  | **（デフォルト値）** Pod 上の**バージョン Annotation**のチェックを無視し、セレクタールールに基づいて直接注入します |
| `true`   | **バージョン Annotation**のチェックを有効にし、一致する Pod にバージョン Annotation が存在する場合のみ注入します             |

#### Annotation タイプの説明 {#annotation-types}

DataKit Operator は 2 種類の Annotation をサポートしており、それぞれ動作が異なります。

**1. 有効化／無効化 Annotation（`check_annotation` の影響を受けません）**
これらの Annotation は、特定の機能を有効または無効にするために使用され、**`check_annotation` 設定の影響を受けません**。

| Annotation                            | 機能説明            | 値             | 優先度   |
| ------------                          | ----------          | ------           | -------- |
| `admission.datakit/enabled`           | すべての注入機能を制御します    | `"true"/"false"` | **最高** |
| `admission.datakit/ddtrace.enabled`   | ddtrace 注入を制御します   | `"true"/"false"` | 中       |
| `admission.datakit/logfwd.enabled`    | logfwd 注入を制御します    | `"true"/"false"` | 中       |
| `admission.datakit/flameshot.enabled` | flameshot 注入を制御します | `"true"/"false"` | 中       |

**2. バージョン Annotation（`check_annotation` の影響を受けます）**
これらの Annotation はコンポーネントのバージョンを指定するために使用され、**`check_annotation` 設定によって制御されます**。

| Annotation                                  | 機能説明                       | 値            |
| --------------------------------------      | ------------------------------ | ------------    |
| `admission.datakit/java-lib.version`        | DDTrace Java Agent のバージョンを指定します   | バージョン文字列      |
| `admission.datakit/python-lib.version`      | DDTrace Python Agent のバージョンを指定します | バージョン文字列      |
| `admission.datakit/java-profiler.version`   | Java Profiler のバージョンを指定します        | バージョン文字列      |
| `admission.datakit/python-profiler.version` | Python Profiler のバージョンを指定します      | バージョン文字列      |
| `admission.datakit/golang-profiler.version` | Golang Profiler のバージョンを指定します      | バージョン文字列      |
| `admission.datakit/logfwd.instances`        | logfwd Sidecar のバージョンを指定します       | JSON 設定文字列 |

#### 注入ロジック {#injection-logic}

**基本ルール**：

- `admission.datakit/enabled:"false"` はすべての注入を拒否します（最高優先度）
- 機能固有の有効化 Annotation（`admission.datakit/ddtrace.enabled: "false"` など）は、その機能の注入を拒否します
- `check_annotation: true` の場合、対応するバージョン Annotation が存在する場合のみ注入します
- `check_annotation: false` の場合、バージョン Annotation のチェックを無視します

**注入条件の比較**：

| 条件                          | `check_annotation: true` | `check_annotation: false` |
|-------------------------------|--------------------------|---------------------------|
| 設定の一致（selector ルール）     | ✓ 必須              | ✓ 必須               |
| 有効化 Annotation が `"false"` ではない        | ✓ 必須              | ✓ 必須               |
| バージョン Annotation が存在する                  | ✓ 必須              | ✗ 無視可能                 |

#### ユースケース例 {#use-case-examples}

1. **一括注入**（`check_annotation: false`）：
   多数の Pod に自動注入する場合に適しており、各 Pod にバージョン Annotation を追加する必要はありません。

2. **詳細な制御**（`check_annotation: true`）：
   バージョンを厳密に制御し、明示的にマークされた Pod のみに注入する場合に適しています。



## サポートされる注入機能 {#supported-operator}

| 機能           | 概要                                                                                      |
| ---            | ---                                                                                       |
| DDtrace Agent  | DDTrace コンポーネントを注入します。詳細は[こちら](operator-ddtrace.md)を参照してください                                        |
| logfwd         | logfwd コンポーネントを注入してコンテナ内のログを収集します。詳細は[こちら](operator-logfwd.md)を参照してください                           |
| Flameshot      | Flameshot コンポーネントを注入してアプリケーションのプロファイリングを動的に収集します。詳細は[こちら](operator-flameshot.md)を参照してください             |
| async-profiler | async-profiler を注入して Java アプリケーションのプロファイリングを定期的に収集します。詳細は[こちら](operator-asyncprofile.md)を参照してください |
| py-spy         | py-spy を注入して Python アプリケーションのプロファイリングを収集します。詳細は[こちら](operator-pyspy.md)を参照してください                  |
| logging        | ログ収集設定を注入します。詳細は[こちら](operator-logging.md)を参照してください                                        |

## 環境変数値の参照 {#downwardapi}

DataKit Operator の `envs` は、プレースホルダーを Kubernetes ネイティブの環境変数値参照に変換できます。このうち、`fieldRef` は [:octicons-tag-24: v1.4.2](operator-changelog.md#cl-1.4.2) 以降でサポートされています。各フィールドについては、Kubernetes の [Downward API](https://kubernetes.io/zh-cn/docs/concepts/workloads/pods/downward-api/#downwardapi-fieldRef)を参照してください。現在、次の形式をサポートしています。

| フィールド                                       | 説明                                            | 例                                 |
| ------:                                    | :------                                         | :------                              |
| `{fieldRef:metadata.name}`                 | Pod の名前                                      | `nginx-123`                          |
| `{fieldRef:metadata.namespace}`            | Pod の namespace                                  | middleware                           |
| `{fieldRef:metadata.uid}`                  | Pod の一意な ID                                   | 12345678-1234-1234-1234-123456789abc |
| `{fieldRef:metadata.annotations['<KEY>']}` | Pod の Annotation `<KEY>` の値                         | metadata.annotations['myannotation'] |
| `{fieldRef:metadata.labels['<KEY>']}`      | Pod のラベル `<KEY>` の値                         | metadata.labels['app']               |
| `{fieldRef:spec.serviceAccountName}`       | Pod のサービスアカウント名                              | default                              |
| `{fieldRef:spec.nodeName}`                 | Pod の実行ノード名                        | node-01                              |
| `{fieldRef:status.hostIP}`                 | Pod が配置されているノードのプライマリ IP アドレス                        | 192.168.1.1                          |
| `{fieldRef:status.hostIPs}`                | status.hostIP のデュアルスタックバージョン                    | ["192.168.1.1", "2001:db8::1"]       |
| `{fieldRef:status.podIP}`                  | Pod のプライマリ IP アドレス                                | 10.0.0.1                             |
| `{resourceFieldRef:limits.cpu}`            | Pod の最初のコンテナの CPU Limit（単位：millicores）   | 500                                  |
| `{resourceFieldRef:limits.memory}`         | Pod の最初のコンテナの Memory Limit（単位：MiB）       | 1024                                 |
| `{resourceFieldRef:requests.cpu}`          | Pod の最初のコンテナの CPU Request（単位：millicores） | 200                                  |
| `{resourceFieldRef:requests.memory}`       | Pod の最初のコンテナの Memory Request（単位：MiB）     | 512                                  |
| `{secretKeyRef:<SECRET_NAME>.<KEY>}`       | Pod が属する namespace 内の Secret key を参照します         | `{secretKeyRef:flameshot-oss.access_key_id}` |

例として、Pod 名が `nginx-123`、namespace が `middleware` の既存の Pod に、環境変数 `POD_NAME` と `POD_NAMESPACE` を注入する場合は、次のように設定します。

```json
{
    "admission_inject_v2": {
        "ddtraces": [
            {
                "namespace_selectors": ["middleware"],
                "language":            "java",
                "image":               "dd-lib-java-init:latest",
                "envs": {
                    "POD_NAME":      "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}"
                }
            }
        ]
    }
}
```

最終的に、この Pod では次の内容を確認できます。

``` shell
$ env | grep POD
POD_NAME=nginx-123
POD_NAMESPACE=middleware
```

<!-- markdownlint-disable MD046 -->
???+ note

    Value プレースホルダーを認識できない場合、通常の文字列として環境変数に追加されます。例えば、`"POD_NAME": "{fieldRef:metadata.PODNAME}"` は誤った記述であり、環境変数では `POD_NAME={fieldRef:metadata.PODNAME}` になります。

### `{resourceFieldRef:*}` に関する重要事項 {#resourcefieldref-important-notes}

`{resourceFieldRef:*}` プレースホルダーは、Pod 内の**最初のコンテナ**のリソース制限（limits）とリクエスト（requests）を参照するために使用します。使用時には、次の点に注意してください。

1. **リソースチェック**：Pod の最初のコンテナに対応するリソース制限またはリクエストが設定されていない場合、このプレースホルダーを使用する環境変数は**注入されません**。例：
   - コンテナに `limits.cpu` が設定されていない場合、`{resourceFieldRef:limits.cpu}` 環境変数は無視されます
   - コンテナに `requests.memory` が設定されていない場合、`{resourceFieldRef:requests.memory}` 環境変数は無視されます

1. **単位**：
   - CPU の単位は **millicores (m)** です。例えば、`500` は 500m（0.5 CPU）を表します
   - メモリの単位は **MiB** です。例えば、`1024` は 1024Mi（1GiB）を表します

1. **最初のコンテナのみをサポート**：`{resourceFieldRef:*}` で参照できるのは Pod 内の最初のコンテナのリソースのみであり、他のコンテナのリソースは参照できません。

1. **使用例**：

```json
{
    "envs": {
        "APP_CPU_LIMIT": "{resourceFieldRef:limits.cpu}",
        "APP_MEMORY_REQUEST": "{resourceFieldRef:requests.memory}"
    }
}
```

1. **確認方法**：注入後の Pod の環境変数を確認することで、リソースプレースホルダーが正しく解析されたかを検証できます。

```shell
kubectl exec <pod-name> -- env | grep APP_
APP_CPU_LIMIT=500
APP_MEMORY_REQUEST=512
```
<!-- markdownlint-enable MD046 -->

### `{secretKeyRef:*}` に関する説明 {#secretkeyref}

DataKit Operator の注入設定では、次の形式を使用して `envs` 内から Kubernetes Secret を参照できます。

```text
{secretKeyRef:<secret-name>.<key>}
```

例えば、Secret `flameshot-oss` に `access_key_id` と `access_key_secret` の 2 つの key が含まれている場合、次のように設定できます。

```json
{
    "envs": {
        "FLAMESHOT_HPROF_UPLOAD_ACCESS_KEY_ID": "{secretKeyRef:flameshot-oss.access_key_id}",
        "FLAMESHOT_HPROF_UPLOAD_ACCESS_KEY_SECRET": "{secretKeyRef:flameshot-oss.access_key_secret}"
    }
}
```

Operator は、これを Kubernetes ネイティブの環境変数参照に変換します。

```yaml
env:
  - name: FLAMESHOT_HPROF_UPLOAD_ACCESS_KEY_ID
    valueFrom:
      secretKeyRef:
        name: flameshot-oss
        key: access_key_id
  - name: FLAMESHOT_HPROF_UPLOAD_ACCESS_KEY_SECRET
    valueFrom:
      secretKeyRef:
        name: flameshot-oss
        key: access_key_secret
```

使用時には、次の点に注意してください。

1. Secret は、注入対象の Pod と同じ namespace に配置する必要があります。Operator は Secret の読み取りやチェックを行いません。Secret はコンテナの起動時に Kubernetes によって解決されます。
1. `<secret-name>` は有効な Kubernetes Secret 名である必要があります。`.` は Secret 名と key の区切り文字として使用されるため、この形式の Secret 名には `.` を含めることはできません。
1. `<key>` は有効な Kubernetes Secret data key である必要があります。長さは 253 文字以下で、英字、数字、`-`、`_`、または `.` のみを使用できます。また、`.`、`..` にすることや、`..` で始めることはできません。
1. Secret または key が存在しない場合、対応する Secret と key が利用可能になるまで、Pod は `CreateContainerConfigError` 状態になります。
1. 認識できない、または検証に失敗した式からは `secretKeyRef` が生成されず、通常の文字列として注入されます。

## FAQ {#faq}

### 特定の Pod への注入を無効にするにはどうすればよいですか？ {#disable-inject}

対象 Pod に Annotation `"admission.datakit/enabled": "false"` を追加すると、その Pod に対するすべての操作が実行されなくなります。この設定が最も高い優先度を持ちます。

### 動作原理を教えてください {#principles}

DataKit-Operator は Kubernetes Admission Controller 機能を使用してリソースを注入します。詳細な仕組みについては、[公式ドキュメント](https://kubernetes.io/zh-cn/docs/reference/access-authn-authz/admission-controllers/){:target="_blank"}を参照してください。

### AWS EKS 環境では何に注意する必要がありますか？ {#aws-eks}

AWS EKS 環境にデプロイすると、DataKit-Operator が動作しない場合があります。セキュリティグループで `9543` ポートを開放する必要があります。

### トラブルシューティングガイド {#debug}

| 問題 | 考えられる原因 | 解決策 |
|--- |--- |---|
| 注入が反映されない | Webhook が正しく設定されていない | `MutatingAdmissionWebhook` と `ValidatingAdmissionWebhook` を確認します |
| イメージの取得に失敗する | イメージアドレスまたは権限の問題 | イメージアドレスを検証し、イメージリポジトリへのアクセス権限を確認します |
| ポートに到達できない | ネットワークまたはセキュリティグループの設定 | `9543` ポートを開放し、ネットワークポリシーを確認します |
