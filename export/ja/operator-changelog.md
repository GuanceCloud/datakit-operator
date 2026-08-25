# Operator 更新履歴

## 1.9.0（リリース予定） {#cl-1.9.0}

- Java、Python、および Node.js アプリケーションへの OpenTelemetry 自動計装の注入をサポート（#98、#101、#102）
- OTel の注入時に `runAsNonRoot: true` が設定され、ゼロ以外の `runAsUser` が明示されていない場合は注入をスキップし、検索可能な warning を記録してアプリケーション Pod の起動失敗を防止（#105）
- Admission Webhook の再入時に DDTrace Java の Agent 起動パラメーターが欠落する可能性がある問題を修正（#100）
- Operator の中国語、英語、日本語、および韓国語ドキュメントを個別に管理（#106）
- 無効または空白の Admission selector では warning を記録し、該当ルールだけを無効にすることで、Pod の一致範囲が意図せず拡大する問題を防止（#87）

## 1.8.10(2026-07-06) {#cl-1.8.10}

- `secretKeyRef` で Kubernetes Secret を参照する環境変数の注入をサポート（#95）

## 1.8.9(2026-06-23) {#cl-1.8.9}

- Cluster API の Pod 照会パフォーマンスを改善し、eBPF 向けの簡略表示をサポート。大規模クラスターでのレスポンスサイズと解析負荷を軽減（#94）

## 1.8.8(2026-05-28) {#cl-1.8.8}

- ddtrace の注入時に `check_annotation` 設定が想定どおりに機能しない場合がある問題を修正（#91）

## 1.8.7(2026-05-14) {#cl-1.8.7}

- Helm Chart と YAML のイメージ tag を更新

## 1.8.6(2026-05-12) {#cl-1.8.6}

- Node.js ddtrace agent の注入をサポート（#89）

## 1.8.5(2026-05-09) {#cl-1.8.5}

- 一部のリソース注入時に元の Pod フィールドが上書きまたは欠落し、Pod が正常に動作しなくなる問題を修正（#88）

## 1.8.4(2026-04-03) {#cl-1.8.4}

- リソース注入時に initContainer の RestartPolicy 設定が失われる可能性がある問題を修正（#86）

## 1.8.3(2026-03-20) {#cl-1.8.3}

- PHP ddtrace agent の注入をサポート（#82）

## 1.8.2(2026-03-10) {#cl-1.8.2}

- 一部の状況で `/logging/configs` API が異常なレスポンスを返す問題を修正（#85）
- Operator の動作状況を調査しやすいように一部のログ出力を改善

## 1.8.1(2026-02-11) {#cl-1.8.1}

- `resourceFieldRef` 形式での環境変数注入をサポート。limits.cpu、limits.memory、requests.cpu、requests.memory など、コンテナのリソース制限値と要求値を参照可能（#84）
- Python ddtrace agent の注入をサポート（#82）
- クラスター内の Pod データを取得するプロキシ API を追加（#79）

## 1.8.0(2026-01-29) {#cl-1.8.0}

- `admission_inject_v2` 設定に `check_annotation` フィールドを追加し、`admission.datakit/java-lib.version` の使用方法との互換性を確保（#81）
- 旧設定 `admission_inject.profiler` による Profiler 注入との互換性を確保（#81）

## 1.7.3(2026-01-08) {#cl-1.7.3}

- ClusterLoggingConfig CRD の `from_beginning_threshold_size` 設定をサポート（#80）

## 1.7.2(2025-12-18) {#cl-1.7.2}

- logfwd の注入時にマウントエラーが発生する可能性がある問題を修正（#78）

## 1.7.1(2025-12-17) {#cl-1.7.1}

- 旧設定との互換性に関する問題を修正（#77）
- `admission_inject_v2` の `namespace_selectors` と `label_selectors` が両方とも空の場合は注入しないように変更（#77）
- flameshot の注入に `enable_prometheus_annotations` 設定を追加し、Prometheus.io Annotations の追加をサポート（#74）
- flameshot のポート設定を、環境変数 `FLAMESHOT_HTTP_LOCAL_ADDR` から `FLAMESHOT_HTTP_LOCAL_PORT` に変更（#75）
- logfwd は CRD 設定ソースを使用できるため、注入時に `log_configs` が必須となっていた問題を修正（#76）

## 1.7.0(2025-12-12) {#cl-1.7.0}

- `admission_inject_v2` 設定を追加して従来の `admission_inject` を置き換え、後方互換性を維持（#73）
- 従来の Profiler 注入に代わる flameshot 注入を追加（#72）
- selector の `namespace_selectors` と `label_selectors` を同時に設定した場合の関係を OR から AND に変更
- Annotation `admission.datakit/logfwd.log_configs` と `admission.datakit/logfwd.volume_paths` のサポートを削除（v1.6.X だけでサポート）
- Annotation `admission.datakit/java-lib.version` のサポートを削除（`admission.datakit/ddtrace.enabled:"false"` による注入の無効化は引き続き可能）

## 1.6.1(2025-12-02) {#cl-1.6.1}

- Kubernetes ClusterLoggingConfig CRD のエラーチェックを改善し、リソースが存在しない場合はログを出力して機能を停止（#71）

## 1.6.0(2025-11-19) {#cl-1.6.0}

- Kubernetes ClusterLoggingConfig CRD をサポートし、ログ収集設定を返す HTTP API を追加（#70）
- 新しい CRD ログ収集設定に対応するよう、logfwd の注入方式を調整および改善（#70）

## 1.5.18(2025-07-15) {#cl-1.5.18}

- ddtrace の注入時に Resources Requests と Resources Limits を手動設定できるように変更（#68）

## 1.5.17(2025-06-03) {#cl-1.5.17}

- リリース image で uos arm64 をサポート

## 1.5.16(2025-04-16) {#cl-1.5.16}

- logging 設定の注入におけるマッチング順序を調整（#64）

## 1.5.15(2025-04-15) {#cl-1.5.15}

- Kubernetes 1.19 へのデプロイ時に namespace を照合できない問題を改善（#63）

## 1.5.14(2025-04-01) {#cl-1.5.14}

- namespace の正規表現を改善し、すべてに一致させる場合は `*` だけを指定すればよく、`.*` は不要になるように変更（#59）

## 1.5.13(2025-03-31) {#cl-1.5.13}

- namespace と labels を正規表現で照合し、DDTrace と Profiler を注入できるように変更（#58）
- namespace と labels を正規表現で照合し、logging 設定を注入できるように変更（#59）

## 1.5.12(2025-02-28) {#cl-1.5.12}

- logfwd と profiler の注入時に Resources Requests と Resources Limits を手動設定できるように変更（#57）

## 1.5.11(2025-02-20) {#cl-1.5.11}

- 対象 Pod への datakit/logs Annotation 設定の追加とディレクトリの自動マウントをサポート（#53）
- logfwd sidecar のディレクトリマウント処理を改善し、既存のマウントをデフォルトで再利用（#55）

## 1.5.10(2024-12-10) {#cl-1.5.10}

- 環境変数 DD_TAGS の注入順序を調整し、先に定義した環境変数の値を常に参照できるように変更（#51）

## 1.5.9(2024-11-25) {#cl-1.5.9}

- 注入イメージの pullPolicy を Always に変更（#50）

## 1.5.8(2024-10-10) {#cl-1.5.8}

- ddtrace Python の注入処理を変更し、イメージと lib を注入せず、基本環境変数だけを追加するように変更（#35）

## 1.5.7(2024-09-18) {#cl-1.5.7}

- v1.5.5 と v1.5.6 で環境変数 DD_TAGS の注入が正しくない問題を修正（#48）
- Pod に Annotations（`admission.datakit/ddtrace.enabled="false"`、`admission.datakit/logfwd.enabled="false"`、`admission.datakit/profiler.enabled="false"`）を追加し、注入機能を個別に無効化できるように変更（#49）

## 1.5.6(2024-09-13) {#cl-1.5.6}

- profiler の volumeMount 注入処理を改善し、同じ path がすでに存在する場合は注入しないように変更（#46）

## 1.5.5(2024-09-02) {#cl-1.5.5}

- ddtrace 環境変数 `DD_TAGS` の注入処理を改善し、元の Pod に `DD_TAGS` がある場合は無視せず追記するように変更（#45）

## 1.5.4(2024-08-27) {#cl-1.5.4}

- ddtrace 注入時の resource を削除し、logfwd と profiler の resource を削減（#44）

## 1.5.3(2024-04-09) {#cl-1.5.3}

- 複数コンテナ環境で ddtrace のマウントファイルが欠落する問題を修正（#42）

## 1.5.2(2024-04-08) {#cl-1.5.2}

- labelSelector に基づく ddtrace の一括注入をサポート（#41）

## 1.5.1(2024-04-07) {#cl-1.5.1}

- ログに現在のバージョンとビルド情報を出力（#41）

## 1.5.0(2024-03-18) {#cl-1.5.0}

- Pod に Annotation `admission.datakit/enabled="false"` を追加し、ddtrace、logfwd、profiler を含むすべての注入を無効化できるように変更（#39）
- 指定した namespace への ddtrace の一括注入をサポート（#38）
- logfwd の注入時に volume を再利用できるようにし、同じパスを複数回マウントした場合のエラーを回避（#34）

## 1.4.3(2023-12-21) {#cl-1.4.3}

- logfwd の注入時に logfiles へワイルドカードパスを指定すると mount エラーになる問題を修正（#31）
- 2個以上のコンテナを持つ Pod への logfwd 注入に失敗し、元の Pod の起動に影響する問題を修正（#32）
- ログ出力を1か所改善

## 1.4.2(2023-09-18) {#cl-1.4.2}

- 注入する環境変数の順序が一定しない問題を修正 (#28)

## 1.4.1(2023-09-15) {#cl-1.4.1}

- logfwd への環境変数注入をサポート (#27)

## 1.4.0(2023-09-13) {#cl-1.4.0}

- Kubernetes Downward API FieldRef 形式による環境変数設定をサポート (#26)
- ddtrace で DD_TAGS をデフォルト追加 (#26)

## 1.3.1(2023-08-24) {#cl-1.3.1}

- デフォルトの Profiler image を更新

## 1.3.0(2023-07-17) {#cl-1.3.0}

- admission 注入の最小単位を Pod に変更。yaml を最新版または datakit-operator-v1.3.0.yaml へ更新する必要があります (#22)
- profiler の注入をサポート (#5)
- 注入する sidecar に Resource Limit を追加 (#20)

## 1.2.1(2023-06-28) {#cl-1.2.1}

- デフォルトのイメージレジストリを変更 (#19)
- Helm の構造を更新

## 1.2.0(2023-06-13) {#cl-1.2.0}

- Datakit Operator の JSON 設定をサポートし、既存の環境変数方式との互換性を維持 (#19)

## 1.0.5(2023-05-11) {#cl-1.0.5}

- 新しい ping API を追加 (#18)
- Datakit の選出専用 API を追加し、Datakit コレクターへのタスク配信を実装 (#15)

## 1.0.4(2023-04-10) {#cl-1.0.4}

- Kubernetes Admission による logfwd プログラムの注入をサポート (#12)
- ddtrace agent の注入時に、環境変数 `DD_AGENT_HOST` と `DD_TRACE_AGENT_PORT` をデフォルト追加 (#14)
- Admission がサポートする resources の一覧を変更し、ネイティブ Pod のサポートを終了 (#12)
- コード構造を変更し、ddtrace と logfwd のユニットテストを追加
- datakit.yaml の構造を改善
- docs ディレクトリとドキュメントを削除し、README に新しいドキュメントリンクを追加

## 1.0.3(2023-03-27) {#cl-1.0.3}

- 英語ドキュメントを追加
- 環境変数による dd-agent イメージアドレスの設定をサポート (#9)
- 複数の環境変数名を改善
- datakit.yaml を変更し、webhook エラーをデフォルトで無視するように設定 (#11)

## 1.0.2(2023-03-09) {#cl-1.0.2}

- CHANGELOG を追加 (#9)
- 証明書の期限切れによるアクセス失敗を修正し、有効期限を延長した自己署名証明書を再生成 (#8)
- image のリリース時に発生する軽微なエラーを修正 (#10)
- yaml のインストール方式を変更し、yaml では namespace を作成せず、ドキュメントに手順を追加 (#7)

## 1.0.1(2022-12-28) {#cl-1.0.1}

- Makefile、Dockerfile、および CI 設定を追加 (#2)
- Kubernetes Admission による ddtrace ファイルと環境変数の注入をサポート (#1)
