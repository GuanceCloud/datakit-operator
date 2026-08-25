# Operator < 1.6.0 logfwd 사용법

이 구성 방식은 DataKit-Operator v1.6.0 이하 버전에서 사용합니다. v1.7.0 버전은 새로운 CRD 구성 방식을 사용하며, v1.6.0 버전에서 도입된 CRD + Annotation 혼합 방식은 더 이상 사용되지 않습니다.

1. 대상 Kubernetes 클러스터에 [DataKit-Operator를 다운로드하고 설치합니다](datakit-operator.md#install).
1. deployment에 지정된 Annotation을 추가하여 logfwd 사이드카를 마운트하도록 설정합니다. Annotation은 template에 추가해야 합니다.
    - key는 모두 `admission.datakit/logfwd.instances`입니다.
    - value는 구체적인 logfwd 구성을 나타내는 JSON 문자열이며, 예시는 다음과 같습니다.

```json
[
    {
        "datakit_addr": "datakit-service.datakit.svc:9533",
        "loggings": [
            {
                "logfiles":      ["<your-logfile-path>"],
                "ignore":        [],
                "storage_index": "<your-storage-index>",
                "source":        "<your-source>",
                "service":       "<your-service>",
                "pipeline":      "<your-pipeline.p>",
                "character_encoding": "",
                "multiline_match": "<your-match>",
                "tags": {}
            },
            {
                "logfiles": ["<your-logfile-path-2>"],
                "source": "<your-source-2>"
            }
        ]
    }
]
```

매개변수에 대한 설명은 [logfwd 구성](../integrations/logfwd.md#config)을 참조하십시오.

- `datakit_addr`은 DataKit logfwdserver 주소입니다.
- `loggings`은 배열로 구성된 주요 설정입니다. [DataKit logging 수집기](../integrations/logging.md)를 참조하십시오.
    - `logfiles` 로그 파일 목록입니다. 절대 경로를 지정할 수 있으며 glob 규칙을 사용한 일괄 지정도 지원합니다. 절대 경로 사용을 권장합니다.
    - `ignore` glob 규칙을 사용하는 파일 경로 필터입니다. 필터 조건 중 하나라도 충족하는 파일은 수집하지 않습니다.
    - `storage_index` 로그 저장 인덱스를 지정합니다.
    - `source` 데이터 소스입니다. 비어 있으면 기본적으로 'default'를 사용합니다.
    - `service` 새로운 tag를 추가합니다. 비어 있으면 기본적으로 $source를 사용합니다.
    - `pipeline` Pipeline 스크립트 경로입니다. 비어 있으면 $source.p를 사용하며, $source.p가 없으면 Pipeline을 사용하지 않습니다. 이 스크립트 파일은 DataKit 측에 있습니다.
    - `character_encoding` 인코딩을 선택합니다. 인코딩이 잘못되면 데이터를 확인할 수 없으므로 기본값인 빈 값으로 두면 됩니다. `utf-8/utf-16le/utf-16le/gbk/gb18030`을 지원합니다.
    - `multiline_match` 여러 줄 매칭입니다. 자세한 내용은 [DataKit 로그 여러 줄 구성](../integrations/logging.md#multiline)을 참조하십시오. JSON 형식이므로 작은따옴표 3개를 사용하는 "이스케이프하지 않는 작성 방식"은 지원하지 않습니다. 정규식 `^\d{4}`은 이스케이프 문자를 추가하여 `^\\d{4}`으로 작성해야 합니다.
    - `tags` 추가 `tag`을 추가합니다. JSON map 형식으로 작성하며, 예시는 `{ "key1":"value1", "key2":"value2" }`입니다.

<!-- markdownlint-disable MD046 -->
???+ note

    logfwd를 주입할 때 DataKit Operator는 기본적으로 동일한 경로의 volume을 재사용하여, 같은 경로의 volume이 존재할 때 발생하는 주입 오류를 방지합니다.

    경로 끝에 슬래시가 있는 경우와 없는 경우는 의미가 다릅니다. 예를 들어 `/var/log`과 `/var/log/`은 서로 다른 경로이므로 재사용할 수 없습니다.
<!-- markdownlint-enable MD046 -->

## 사용 예 {#datakit-operator-inject-logfwd-example}

다음은 shell을 사용하여 파일에 데이터를 계속 쓰고 해당 파일을 수집하도록 구성한 Deployment 예시입니다.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
    name: logging-deployment
    labels:
    app: logging
spec:
    replicas: 1
    selector:
    matchLabels:
        app: logging
    template:
    metadata:
        labels:
        app: logging
        annotations:
        admission.datakit/logfwd.instances: '[{"datakit_addr":"datakit-service.datakit.svc:9533","loggings":[{"logfiles":["/var/log/log-test/*.log"],"source":"deployment-logging","tags":{"key01":"value01"}}]}]'
    spec:
        containers:
        - name: log-container
        image: busybox
        args: [/bin/sh, -c, 'mkdir -p /var/log/log-test; i=0; while true; do printf "$(date "+%F %H:%M:%S") [%-8d] Bash For Loop Examples.\\n" $i >> /var/log/log-test/1.log; i=$((i+1)); sleep 1; done']
```

yaml 파일을 사용하여 리소스를 생성합니다.

```shell
$ kubectl apply -f logging.yaml
...
```

다음과 같이 확인합니다.

```shell
$ kubectl get pod

NAME                                   READY   STATUS    RESTARTS      AGE
logging-deployment-5d48bf9995-vt6bb    1/1     Running   0             4s

$ kubectl get pod logging-deployment-5d48bf9995-vt6bb -o=jsonpath={.spec.containers\[\*\].name}
log-container datakit-logfwd
```

마지막으로 <<<custom_key.brand_name>>> 로그 플랫폼에서 로그 수집 여부를 확인할 수 있습니다.
