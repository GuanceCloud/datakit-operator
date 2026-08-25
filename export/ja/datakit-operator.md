# DataKit Operator

---

:material-kubernetes:

---

DataKit Operator は Kubernetes Admission Webhook を介して、新規 Pod にトレース、ログ収集、およびパフォーマンス分析コンポーネントを注入し、クラスター内の Pod を照会するインターフェースも提供します。

## 概要 {#overview}

DataKit Operator は次の機能を提供します。

| 機能 | 説明 |
| --- | --- |
| DDTrace 自動注入 | Java、Python、PHP、および Node.js をサポート |
| OpenTelemetry 自動注入 | v1.9.0 以降で Java、Python、および Node.js をサポート |
| logfwd 注入 | コンテナの標準出力に書き込まれないファイルログを Sidecar で収集 |
| Flameshot 注入 | アプリケーションの Profiling データを動的に収集 |
| Profiler 注入 | 旧方式の async-profiler や py-spy などの注入に対応 |
| Logging 設定の注入 | `datakit/logs` Annotation と対応するファイルボリュームのマウントを追加 |
| Cluster API | クラスター内の Pod データをプロキシ経由で照会し、DataKit から API Server への直接アクセスによる負荷を軽減 |

注入は Pod の `CREATE` 時に行われ、実行中の Pod は変更されません。Operator の設定、注入イメージ、またはワークロードの Annotation を変更した場合は、Pod を再作成すると反映されます。

Admission は fail-open 方式です。注入に失敗すると Operator はログを記録しますが、アプリケーション Pod の作成は妨げません。デプロイマニフェストの webhook でも `failurePolicy: Ignore` を使用します。

## 前提条件 {#prerequisites}

- Kubernetes が `admissionregistration.k8s.io/v1` をサポートしていること。Kubernetes v1.24 以降を推奨します。
- クラスターで `MutatingAdmissionWebhook` Admission Controller が有効になっていること。
- クラスターノードから設定済みイメージを pull できること。オフライン環境では、事前にイメージをプライベートレジストリへ同期してください。

## インストール {#install}

<!-- markdownlint-disable MD046 -->
=== "Deployment"

    [*datakit-operator.yaml*](https://static.<<<custom_key.brand_main_domain>>>/datakit-operator/datakit-operator.yaml){:target="_blank"} をダウンロードしてインストールします。

    ```shell
    kubectl create namespace datakit
    wget https://static.<<<custom_key.brand_main_domain>>>/datakit-operator/datakit-operator.yaml
    kubectl apply -f datakit-operator.yaml
    kubectl get pod -n datakit
    ```

    Pod が正常に起動すると、ステータスは `Running` になります。

    ```text
    NAME                                READY   STATUS    RESTARTS   AGE
    datakit-operator-f948897fb-5w5nm    1/1     Running   0          15s
    ```

=== "Helm"

    Helm 3.0 以降が必要です。

    ```shell
    helm install datakit-operator datakit-operator \
        <<<% if custom_key.brand_key == 'guance' -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/datakit-operator \
        <<<% else -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/truewatch \
        <<<% endif -%>>>
        -n datakit --create-namespace
    ```

    デプロイ状態を確認します。

    ```shell
    helm -n datakit list
    ```

    アップグレードします。

    ```shell
    helm -n datakit get values datakit-operator -a -o yaml > values.yaml
    helm upgrade datakit-operator datakit-operator \
        <<<% if custom_key.brand_key == 'guance' -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/datakit-operator \
        <<<% else -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/truewatch \
        <<<% endif -%>>>
        -n datakit \
        -f values.yaml
    ```

    アンインストールします。

    ```shell
    helm uninstall datakit-operator -n datakit
    ```

???+ attention

    - Operator 本体とデプロイマニフェストは、対応する組み合わせで使用する必要があります。Operator のアップグレード時には、YAML または Helm Chart も同時に更新してください。
    - `InvalidImageName` が表示される場合やイメージの pull に失敗する場合は、イメージアドレス、レジストリ権限、およびノードのネットワークを確認してください。
<!-- markdownlint-enable MD046 -->

### 設定について {#jsonconfig}

Operator の設定は JSON 形式です。通常、デプロイマニフェストでは設定を ConfigMap に保存し、`ENV_JSON_CONFIG` 環境変数から読み込みます。

v1.8.0 以降では `admission_inject_v2` を使用します。

```json
{
    "server_listen": "0.0.0.0:9543",
    "log_level": "info",
    "admission_inject_v2": {
        "ddtraces": [],
        "otels": [],
        "logfwds": [],
        "flameshots": [],
        "profilers": []
    },
    "admission_mutate": {
        "loggings": []
    }
}
```

旧方式の `admission_inject` 設定との互換性も維持されています。旧設定で有効な `ddtrace`、`logfwd`、または `profiler` は、対応する v2 ルールをそれぞれ上書きします。アップグレード時に、有効な設定を両方で管理しないでください。

## Cluster API {#cluster-api}

DataKit Operator [:octicons-tag-24: v1.8.1](operator-changelog.md#cl-1.8.1) 以降では Cluster API を提供します。この API は Operator の Pod informer キャッシュを使用して Pod データをプロキシ経由で照会し、各 DataKit インスタンスから API Server への直接アクセスによる負荷を軽減します。

Cluster API はデフォルトで有効です。Operator は起動時に ServiceAccount の Pod 読み取り権限を確認します。権限が不足している場合やキャッシュの同期に失敗した場合、関連ルートは登録されませんが、その他の機能は引き続き動作します。必要な最小権限は次のとおりです。

```yaml
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
```

この API は Operator Service を使用し、デフォルトのアドレスは `https://datakit-operator.datakit.svc:443` です。

| API | 説明 |
| --- | --- |
| `/v1/cluster/api/v1/pods` | すべての Pod を照会 |
| `/v1/cluster/api/v1/namespaces/{namespace}/pods` | 指定した Namespace 内の Pod を照会 |
| `/v1/cluster/api/v1/namespaces/{namespace}/pods/{name}` | 指定した Pod を照会 |

[:octicons-tag-24: v1.8.9](operator-changelog.md#cl-1.8.9) 以降では、`view=ebpf-v1` を使用して eBPF 向けに簡略化された Pod 構造を取得できます。

```shell
curl -k "https://datakit-operator.datakit.svc:443/v1/cluster/api/v1/pods?view=ebpf-v1"
```

## 注入ルール {#datakit-operator-inject}

各注入設定では、まず selector によって Pod が一致する必要があります。Annotation では注入を拒否でき、`check_annotation` が有効な場合はさらに対象を絞り込めますが、selector とは独立して注入を開始することはできません。

### Selector の設定 {#selectors-injection}

`namespace_selectors` と `label_selectors` は、どちらも配列です。

- 同じ配列内の複数の selector は OR 条件で評価されます。
- 両方の配列を設定した場合、namespace と label の両方が一致する必要があります。
- 注入ルールで一方の条件だけを設定した場合は、その条件だけで照合します。どちらも設定していない場合は現在のルールを無効にし、後続ルールの確認を続けます。
- Logging mutation ルールでは両方の条件が必要です。どちらか一方がない場合は現在のルールを無効にします。

`namespace_selectors` には Go の正規表現を使用します。文字列 `"*"` はすべての Namespace に一致する省略表記です。完全一致には `^` と `$` の使用を推奨します。`label_selectors` には Kubernetes Label Selector 構文を使用し、`=`、`==`、`!=` では glob マッチングも利用できます。

空文字列、空白のみの値、無効な Namespace 正規表現、および無効または制約が空の Label selector は無効な項目です。Operator は起動時に warning を記録して各無効項目を無視します。設定済みの条件に有効な項目が残らない場合は現在のルールを無効にし、後続ルールの確認を続けます。すべての Namespace に明示的に一致させるには `"*"` を使用してください。

次のルールは、`production` Namespace 内で `admission.datakit/ddtrace-language=java` ラベルを持つ Pod だけに一致します。

```json
{
    "namespace_selectors": ["^production$"],
    "label_selectors": ["admission.datakit/ddtrace-language=java"]
}
```

同じ Pod が複数の DDTrace または OTel ルールに一致する場合があります。Operator は設定順に、annotation 条件を満たす最初のルールを選択します。選択後に言語、イメージ、その他の設定でエラーが発生しても、後続ルールへはフォールバックしません。複数言語のルールには、相互排他的なラベルを使用してください。

DDTrace と OTel の両方に一致する場合は DDTrace が優先され、DDTrace の注入に失敗しても OTel へはフォールバックしません。

### Annotation の設定 {#annotation-injection}

Annotation は Pod、または Deployment などのコントローラーの `.spec.template.metadata.annotations` に追加します。

| Annotation | 用途 |
| --- | --- |
| `admission.datakit/enabled` | Operator による Pod のすべての変更を制御。最優先 |
| `admission.datakit/ddtrace.enabled` | DDTrace の注入を制御 |
| `admission.datakit/otel.enabled` | OTel の注入を制御 |
| `admission.datakit/logfwd.enabled` | logfwd の注入を制御 |
| `admission.datakit/flameshot.enabled` | Flameshot の注入を制御 |
| `admission.datakit/profiler.enabled` | 旧方式の Profiler 注入を制御 |

これらのスイッチ用 Annotation は寛容に解釈されます。`false` として解析できる値だけが該当機能を無効にし、Annotation がない場合や解析できない場合は `true` として扱われます。明示的に `true` を設定しても、Pod は対応する設定ルールに一致する必要があります。

```yaml
metadata:
  annotations:
    admission.datakit/ddtrace.enabled: "false"
    admission.datakit/otel.enabled: "true"
```

### `check_annotation` {#check-annotation-config}

DDTrace、OTel、logfwd、および旧方式の Profiler ルールは `check_annotation` をサポートします。

| 値 | 動作 |
| --- | --- |
| `false` | デフォルト。selector の一致後に、バージョンまたは旧方式の設定 annotation を要求しません |
| `true` | selector の一致に加え、そのルールに対応する annotation が必要です |

対応関係は次のとおりです。

| 機能 | `check_annotation: true` の場合に必要な Annotation |
| --- | --- |
| DDTrace | `admission.datakit/<language>-lib.version` |
| OTel | `admission.datakit/otel-<language>-lib.version` |
| Profiler | `admission.datakit/<language>-profiler.version` |
| logfwd | `admission.datakit/logfwd.instances` |

`check_annotation: true` で、Pod に対応するバージョン annotation が指定されている場合、DDTrace、OTel、および Profiler はルール内の `image` の tag を置き換えますが、イメージのレジストリと名前は変更しません。機能スイッチ用 Annotation は `check_annotation` に関係なく常に有効です。

## 対応する注入機能 {#supported-operator}

| 機能 | ドキュメント |
| --- | --- |
| DDTrace | [DDTrace 自動注入](operator-ddtrace.md) |
| OpenTelemetry | [OpenTelemetry 自動注入](operator-otel.md) |
| logfwd | [logfwd Sidecar の注入](operator-logfwd.md) |
| Flameshot | [Flameshot の注入](operator-flameshot.md) |
| async-profiler | [旧方式の Java Profiler 注入](operator-asyncprofile.md) |
| py-spy | [旧方式の Python Profiler 注入](operator-pyspy.md) |
| Logging | [ログ収集設定の注入](operator-logging.md) |

## 環境変数値の参照 {#downwardapi}

注入ルールの `envs` にはリテラル値を指定できるほか、プレースホルダーを Kubernetes ネイティブの `fieldRef`、`resourceFieldRef`、および `secretKeyRef` に変換できます。

| 形式 | 説明 |
| --- | --- |
| `{fieldRef:metadata.name}` | Pod 名 |
| `{fieldRef:metadata.namespace}` | Pod の Namespace |
| `{fieldRef:metadata.uid}` | Pod UID |
| `{fieldRef:metadata.annotations['<KEY>']}` | 指定した Pod Annotation |
| `{fieldRef:metadata.labels['<KEY>']}` | 指定した Pod Label |
| `{fieldRef:spec.serviceAccountName}` | ServiceAccount 名 |
| `{fieldRef:spec.nodeName}` | ノード名 |
| `{fieldRef:status.hostIP}` | ノードのプライマリ IP |
| `{fieldRef:status.hostIPs}` | ノードのデュアルスタック IP |
| `{fieldRef:status.podIP}` | Pod のプライマリ IP |
| `{resourceFieldRef:limits.cpu}` | 最初のアプリケーションコンテナの CPU limit。単位は millicore |
| `{resourceFieldRef:limits.memory}` | 最初のアプリケーションコンテナのメモリー limit。単位は MiB |
| `{resourceFieldRef:requests.cpu}` | 最初のアプリケーションコンテナの CPU request。単位は millicore |
| `{resourceFieldRef:requests.memory}` | 最初のアプリケーションコンテナのメモリー request。単位は MiB |
| `{secretKeyRef:<SECRET_NAME>.<KEY>}` | Pod と同じ Namespace にある Secret の key |

環境変数は設定順を維持します。後続の値から Kubernetes の `$(VAR)` 構文を使用して、先に定義した変数を参照できます。

```json
{
    "envs": {
        "POD_NAME": "{fieldRef:metadata.name}",
        "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
        "RESOURCE_TAGS": "pod_name=$(POD_NAME),pod_namespace=$(POD_NAMESPACE)"
    }
}
```

認識できないプレースホルダーは、通常の文字列として注入されます。`resourceFieldRef` が参照するのは最初のアプリケーションコンテナだけです。そのコンテナに対応する request または limit が宣言されていない場合、環境変数は注入されません。

### `{secretKeyRef:*}` {#secretkeyref}

Secret 参照の形式は次のとおりです。

```text
{secretKeyRef:<secret-name>.<key>}
```

例：

```json
{
    "envs": {
        "ACCESS_KEY_ID": "{secretKeyRef:flameshot-oss.access_key_id}",
        "ACCESS_KEY_SECRET": "{secretKeyRef:flameshot-oss.access_key_secret}"
    }
}
```

Operator は Secret を読み取りません。参照はコンテナの起動時に Kubernetes が解決します。Secret はアプリケーション Pod と同じ Namespace に配置する必要があります。Secret または key が存在しない場合、Pod は `CreateContainerConfigError` になります。`.` は Secret 名と key の区切りに使用されるため、この形式では Secret 名に `.` を含めることはできません。

## FAQ {#faq}

### 特定の Pod に対するすべての変更を無効にするには {#disable-inject}

Pod テンプレートに次の設定を追加します。

```yaml
admission.datakit/enabled: "false"
```

### 注入が反映されない場合 {#debug}

次の順序で確認します。

1. 設定の更新後に再作成された新しい Pod であること。
1. Namespace と Label が同じルールに一致していること。
1. `admission.datakit/enabled` および各機能のスイッチ用 Annotation が `false` ではないこと。
1. `check_annotation: true` の場合、正しいバージョンまたは設定 annotation が存在すること。
1. Operator のログに、selector、イメージ、環境変数、またはセキュリティコンテキストの競合に関する warning がないこと。

### AWS EKS 環境での注意事項 {#aws-eks}

EKS コントロールプレーンから Operator webhook の `9543` ポートへアクセスできる必要があります。注入されない場合は、クラスターのセキュリティグループとネットワークポリシーで、この方向の通信が許可されていることを確認してください。
