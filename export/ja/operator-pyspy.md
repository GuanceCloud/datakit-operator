# DataKit Operator による Python プロファイリングの注入

## 前提条件 {#prerequisites}

- 現在、Python 公式インタープリター（CPython）のみをサポートしています。

お使いの [Pod コントローラー](https://kubernetes.io/docs/concepts/workloads/controllers/){:target="_blank"} リソース設定ファイルの
`.spec.template.metadata.annotations` ノード配下に次のアノテーションを追加し、そのリソース設定ファイルを適用します。
DataKit-Operator は、対応する Pod 内に `datakit-profiler` という名前のコンテナを自動的に作成し、プロファイリングを支援します。

> **アノテーションの使用方法**：`check_annotation` 設定がバージョンアノテーションの動作に与える影響、および各種アノテーションの詳細については、[Annotation 設定による注入](datakit-operator.md#annotation-injection)および [`check_annotation` 設定項目の説明](datakit-operator.md#check-annotation-config)を参照してください。

以下では、"movies-python" という名前の `Deployment` リソース設定ファイルを例に説明します。

```yaml hl_lines="17"
apiVersion: apps/v1
kind: Deployment
metadata:
  name: movies-python
  labels:
    app: movies-python
spec:
  replicas: 1
  selector:
    matchLabels:
      app: movies-python
  template:
    metadata:
      name: movies-python
      labels:
        app: movies-python
      annotations:
        admission.datakit/python-profiler.version: {{.K8sProfilersPySpyVersion}} # <-- add annotation here
    spec:
      containers:
        - name: movies-python
          image: zhangyicloud/movies-python:1.2.3
          imagePullPolicy: Always
          command:
            - "gunicorn"
            - "-w"
            - "4"
            - "--bind"
            - "0.0.0.0:8080"
            - "app:app"
```

リソース設定を適用し、反映されていることを確認します：

```shell
$ kubectl apply -f deployment-movies-python.yaml

$ kubectl get pods | grep movies-python
movies-python-78b6cf55f-ptzxf   2/2     Running   0          64s


$ kubectl describe pod movies-python-78b6cf55f-ptzxf | grep datakit-profiler
      /app/datakit-profiler from datakit-profiler-volume (rw)
  datakit-profiler:
      /app/datakit-profiler from datakit-profiler-volume (rw)
  datakit-profiler-volume:
  Normal  Created    98s   kubelet            Created container datakit-profiler
  Normal  Started    97s   kubelet            Started container datakit-profiler
```

数分待つと、<<<custom_key.brand_name>>> コンソールの [APM-プロファイリング](https://console.<<<custom_key.brand_main_domain>>>/tracing/profile){:target="_blank"} ページでアプリケーションパフォーマンスデータを確認できます。

<!-- markdownlint-disable MD046 -->
???+ note

    - デフォルトでは、コマンド `ps -e -o pid,cmd --no-headers | grep -v grep | grep "python" | head -n 20` を使用してコンテナ内の `Python` プロセスを検索します。パフォーマンス上の理由から、データを収集するプロセスは最大 20 個です。

    - `datakit-operator.yaml` 設定ファイル内の ConfigMap `datakit-operator-config` 配下の環境変数を変更することで、プロファイリングの動作を設定できます。

    | 環境変数              | 説明                                                                                                                                               | デフォルト値                        |
    | ----                  | --                                                                                                                                                 | -----                         |
    | `DK_PROFILE_SCHEDULE` | プロファイリングの実行スケジュール。Linux [Crontab](https://man7.org/linux/man-pages/man5/crontab.5.html){:target="_blank"} と同じ構文を使用します（例：`*/10 * * * *`） | `0 * * * *`（1 時間ごとに 1 回スケジュール） |
    | `DK_PROFILE_DURATION` | 1 回のプロファイリングの継続時間（秒）                                                                                                                  | 240（4 分）                 |


    - データを確認できない場合は、`datakit-profiler` コンテナに入り、該当するログを確認してトラブルシューティングできます：

    ```shell
    $ kubectl exec -it movies-python-78b6cf55f-ptzxf -c datakit-profiler -- bash
    $ tail -n 2000 log/main.log
    ```
<!-- markdownlint-enable MD046 -->