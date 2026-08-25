# DataKit Operator によるログ収集設定の注入

DataKit Operator は、指定した Pod に DataKit Logging の収集に必要な設定を自動的に追加できます。これには、`datakit/logs` アノテーションと、対応するファイルパスの volume/volumeMount が含まれ、手動設定の煩雑な手順を簡素化します。これにより、ユーザーは各 Pod の設定を手動で変更することなく、ログ収集機能を自動的に有効化できます。

以下は、DataKit Operator の `admission_mutate` 設定を使用して、ログ収集設定を自動的に注入する方法を示す設定例です。

```json
{
    "server_listen": "0.0.0.0:9543",
    "log_level":     "info",
    "admission_inject": {
        # その他の設定
    },
    "admission_mutate": {
        "loggings": [
            {
                "namespace_selectors": ["middleware"],
                "label_selectors":     ["app=logging"],
                "config": "[{\"disable\":false,\"type\":\"file\",\"path\":\"/tmp/opt/**/*.log\",\"source\":\"logging-tmp\"}]"
            }
        ]
    }
}
```

`admission_mutate.loggings`：複数のログ収集設定を含むオブジェクト配列です。各ログ設定には、以下のフィールドが含まれます。

- `namespace_selectors`：条件に一致する Pod が属する Namespace を限定します。複数の Namespace を設定でき、Pod が選択されるには少なくとも 1 つの Namespace に一致する必要があります。`label_selectors` とは OR の関係です。
- `label_selectors`：条件に一致する Pod の label を限定します。Pod が選択されるには、少なくとも 1 つの label selector に一致する必要があります。`namespace_selectors` とは OR の関係です。
- `config`：Pod のアノテーションに追加される JSON 文字列です。アノテーションの Key は `datakit/logs` です。この Key がすでに存在する場合、上書きも重複追加もされません。この設定により、DataKit にログの収集方法を指定します。

DataKit Operator は `config` 設定を自動的に解析し、その中のパス（`path`）に基づいて、Pod に対応する volume と volumeMount を作成します。

上記の DataKit Operator 設定を例にすると、Pod の Namespace が `middleware` であるか、Labels が `app=logging` に一致する場合、Pod にアノテーションとマウントを追加します。例：

```yaml hl_lines="5"
apiVersion: v1
kind: Pod
metadata:
  annotations:
    datakit/logs: '[{"disable":false,"type":"file","path":"/tmp/opt/**/*.log","source":"logging-tmp"}]'
  labels:
    app: logging
  name: logging-test
  namespace: default
spec:
  containers:
  - args:
    - |
      mkdir -p /tmp/opt/log1;
      i=1;
      while true; do
        echo "Writing logs to file ${i}.log";
        for ((j=1;j<=10000000;j++)); do
          echo "$(date +'%F %H:%M:%S')  [$j]  Bash For Loop Examples. Hello, world! Testing output." >> /tmp/opt/log1/file_${i}.log;
          sleep 1;
        done;
        echo "Finished writing 5000000 lines to file_${i}.log";
        i=$((i+1));
      done
    command:
    - /bin/bash
    - -c
    - --
    image: pubrepo.<<<custom_key.brand_main_domain>>>/base/ubuntu:18.04
    imagePullPolicy: IfNotPresent
    name: demo
    volumeMounts:
    - mountPath: /tmp/opt
      name: datakit-logs-volume-0
  volumes:
  - emptyDir: {}
    name: datakit-logs-volume-0
```

この Pod には `app=logging` label があり、条件に一致します。そのため、DataKit Operator は `datakit/logs` アノテーションを追加し、パス `/tmp/opt` を EmptyDir としてマウントします。

DataKit のログ収集機能が Pod を検出すると、`datakit/logs` の内容に基づいてカスタム収集を実行します。
