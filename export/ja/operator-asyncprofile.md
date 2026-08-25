# DataKit Operator による async-profiler の注入

## 前提条件 {#async-profiler-prerequisites}

- クラスターに [DataKit](https://docs.<<<custom_key.brand_main_domain>>>/datakit/datakit-daemonset-deploy/){:target="_blank"} がインストールされています。
- [profile を有効化](https://docs.<<<custom_key.brand_main_domain>>>/datakit/datakit-daemonset-deploy/#using-k8-env){:target="_blank"}し、コレクターを起動しています。
- Linux カーネルパラメーター [kernel.perf_event_paranoid](https://www.kernel.org/doc/Documentation/sysctl/kernel.txt){:target="_blank"} の値が 2 以下に設定されています。

<!-- markdownlint-disable MD046 -->
???+ note

    `async-profiler` は [`perf_events`](https://perf.wiki.kernel.org/index.php/Main_Page){:target="_blank"} ツールを使用して Linux カーネルのコールスタックを取得します。非特権プロセスで利用するには、対応するカーネル設定が必要です。次のコマンドを使用してカーネルパラメーターを変更できます。
    ```shell
    $ sudo sysctl kernel.perf_event_paranoid=1
    $ sudo sysctl kernel.kptr_restrict=0
    # または
    $ sudo sh -c 'echo 1 >/proc/sys/kernel/perf_event_paranoid'
    $ sudo sh -c 'echo 0 >/proc/sys/kernel/kptr_restrict'
    ```
<!-- markdownlint-enable MD046 -->
## インジェクション設定 {#annotation-injection}

使用する [Pod コントローラー](https://kubernetes.io/docs/concepts/workloads/controllers/){:target="_blank"} のリソース設定ファイルにある
`.spec.template.metadata.annotations` ノードの下に次のアノテーションを追加し、そのリソース設定ファイルを適用します。
DataKit-Operator は、対応する Pod 内に `datakit-profiler` という名前のコンテナを自動的に作成し、プロファイリングを補助します。

> **アノテーションの使用方法**：`check_annotation` 設定がバージョンアノテーションの動作に与える影響、および各アノテーションの詳細については、[Annotation のインジェクション設定](datakit-operator.md#annotation-injection)および [`check_annotation` 設定項目の説明](datakit-operator.md#check-annotation-config)を参照してください。

次の Deployment リソース設定ファイルを例に説明します。

```yaml hl_lines="17"
kind: Deployment
metadata:
  name: movies-java
  labels:
    app: movies-java
spec:
  replicas: 1
  selector:
    matchLabels:
      app: movies-java
  template:
    metadata:
      name: movies-java
      labels:
        app: movies-java
      annotations:
        admission.datakit/java-profiler.version: "{{.K8sProfilersAsyncProfileVersion}}" # <-- add annotation here
    spec:
      containers:
        - name: movies-java
          image: your/app:v1.2.3
          imagePullPolicy: IfNotPresent
          securityContext:
            seccompProfile:
              type: Unconfined
          env:
            - name: JAVA_OPTS
              value: ""

      restartPolicy: Always
```

設定ファイルを適用し、反映されていることを確認します。

```shell
$ kubectl apply -f deployment-movies-java.yaml

$ kubectl get pods | grep movies-java
movies-java-784f4bb8c7-59g6s   2/2     Running   0          47s

$ kubectl describe pod movies-java-784f4bb8c7-59g6s | grep datakit-profiler
      /app/datakit-profiler from datakit-profiler-volume (rw)
  datakit-profiler:
      /app/datakit-profiler from datakit-profiler-volume (rw)
  datakit-profiler-volume:
  Normal  Created    12m   kubelet            Created container datakit-profiler
  Normal  Started    12m   kubelet            Started container datakit-profiler
```

数分待つと、<<<custom_key.brand_name>>>コンソールの [アプリケーションパフォーマンスモニタリング（APM）-プロファイリング](https://console.<<<custom_key.brand_main_domain>>>/tracing/profile){:target="_blank"} ページでアプリケーションパフォーマンスデータを確認できます。

<!-- markdownlint-disable MD046 -->
???+ note

    - デフォルトでは、コマンド `jps -q -J-XX:+PerfDisableSharedMem | head -n 20` を使用してコンテナ内の JVM プロセスを検索します。パフォーマンスへの影響を考慮し、データを収集するプロセスは最大 20 個です。

    - `datakit-operator.yaml` 設定ファイル内の `datakit-operator-config` 配下にある環境変数を変更することで、プロファイリングの動作を設定できます。


    | 環境変数              | 説明                                                                                                                                               | デフォルト値                        |
    | ----                  | --                                                                                                                                                 | -----                         |
    | `DK_PROFILE_SCHEDULE` | プロファイリングの実行スケジュール。Linux [Crontab](https://man7.org/linux/man-pages/man5/crontab.5.html){:target="_blank"} と同じ構文を使用します（例：`*/10 * * * *`） | `0 * * * *`（1 時間ごとに 1 回実行） |
    | `DK_PROFILE_DURATION` | プロファイリング 1 回あたりの継続時間（秒）                                                                                                                  | 240（4 分）                 |


    - データを確認できない場合は、`datakit-profiler` コンテナに入り、対応するログを確認して原因を調査できます。

    ```shell
    $ kubectl exec -it movies-java-784f4bb8c7-59g6s -c datakit-profiler -- bash
    $ tail -n 2000 log/main.log
    ```
<!-- markdownlint-enable MD046 -->