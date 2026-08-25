# DataKit Operator 로그 수집 설정 주입

DataKit Operator는 지정된 Pod에 DataKit Logging 수집에 필요한 설정을 자동으로 추가할 수 있습니다. 여기에는 `datakit/logs` 어노테이션과 해당 파일 경로의 volume/volumeMount가 포함되며, 번거로운 수동 설정 절차를 간소화합니다. 이를 통해 사용자는 각 Pod 설정에 수동으로 개입하지 않고도 로그 수집 기능을 자동으로 활성화할 수 있습니다.

다음은 DataKit Operator의 `admission_mutate` 설정을 통해 로그 수집 설정을 자동으로 주입하는 방법을 보여 주는 설정 예시입니다.

```json
{
    "server_listen": "0.0.0.0:9543",
    "log_level":     "info",
    "admission_inject": {
        # 기타 설정
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

`admission_mutate.loggings`: 여러 로그 수집 설정을 포함하는 객체 배열입니다. 각 로그 설정에는 다음 필드가 포함됩니다.

- `namespace_selectors`: 조건에 부합하는 Pod가 위치한 Namespace를 제한합니다. 여러 Namespace를 설정할 수 있으며, Pod가 선택되려면 하나 이상의 Namespace와 일치해야 합니다. `label_selectors`와는 “또는” 관계입니다.
- `label_selectors`: 조건에 부합하는 Pod의 label을 제한합니다. Pod가 선택되려면 하나 이상의 label selector와 일치해야 합니다. `namespace_selectors`와는 “또는” 관계입니다.
- `config`: Pod의 어노테이션에 추가되는 JSON 문자열이며, 어노테이션의 Key는 `datakit/logs`입니다. 해당 Key가 이미 존재하면 덮어쓰거나 중복으로 추가하지 않습니다. 이 설정은 DataKit에 로그 수집 방법을 지정합니다.

DataKit Operator는 `config` 설정을 자동으로 파싱하고, 설정에 포함된 경로(`path`)에 따라 Pod에 해당 volume과 volumeMount를 생성합니다.

위 DataKit Operator 설정을 예로 들면, Pod의 Namespace가 `middleware`이거나 Labels가 `app=logging`와 일치하면 Pod에 어노테이션과 마운트를 추가합니다. 예시는 다음과 같습니다.

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

이 Pod에는 `app=logging` label이 있어 조건과 일치하므로, DataKit Operator가 `datakit/logs` 어노테이션을 추가하고 `/tmp/opt` 경로를 EmptyDir로 마운트합니다.

DataKit 로그 수집이 Pod를 감지하면 `datakit/logs` 내용에 따라 맞춤형 수집을 수행합니다.
