# v1.6.0 이하 버전의 logfwd 사용법

이 페이지는 레거시 배포 유지 관리용입니다. DataKit Operator v1.7.0 이상에서는 [현재 logfwd 주입 방식](operator-logfwd.md)을 사용하고 Annotation 기반 정적 구성을 새로 추가하지 마십시오.

이전 버전에서는 `admission.datakit/logfwd.instances` Annotation으로 사이드카를 활성화하는 동시에 수집 작업을 제공합니다. Annotation은 Deployment의 `.spec.template.metadata.annotations`에 추가해야 하며, 값은 JSON 배열 문자열입니다.

```json
[
    {
        "datakit_addr": "datakit-service.datakit.svc:9533",
        "loggings": [
            {
                "logfiles": ["/var/log/app/*.log"],
                "ignore": [],
                "source": "app",
                "service": "checkout",
                "pipeline": "app.p",
                "character_encoding": "",
                "multiline_match": "^\\d{4}-\\d{2}-\\d{2}",
                "tags": {
                    "env": "production"
                }
            }
        ]
    }
]
```

주요 필드는 다음과 같습니다.

| 필드 | 설명 |
| --- | --- |
| `datakit_addr` | DataKit `logfwdserver` 주소 |
| `logfiles` | 파일 절대 경로 배열. glob을 지원합니다. |
| `ignore` | 제외할 파일 경로 배열. glob을 지원합니다. |
| `source` | 로그 소스 |
| `service` | 서비스 이름. 비어 있으면 `source`를 사용합니다. |
| `pipeline` | DataKit의 Pipeline 파일 이름 |
| `character_encoding` | 문자 인코딩. 일반적으로 비워 두면 자동으로 감지합니다. |
| `multiline_match` | 여러 줄 로그의 첫 줄 정규식. JSON에서는 백슬래시를 이스케이프해야 합니다. |
| `tags` | 로그에 추가할 태그 |

## Deployment 예시 {#datakit-operator-inject-logfwd-example}

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: logging-demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: logging-demo
  template:
    metadata:
      labels:
        app: logging-demo
      annotations:
        admission.datakit/logfwd.instances: '[{"datakit_addr":"datakit-service.datakit.svc:9533","loggings":[{"logfiles":["/var/log/app/*.log"],"source":"app"}]}]'
    spec:
      containers:
        - name: app
          image: busybox:1.36
          command: ["sh", "-c"]
          args:
            - mkdir -p /var/log/app; while true; do date >> /var/log/app/app.log; sleep 1; done
```

생성한 후 다음 명령을 실행합니다.

```shell
kubectl get pod -l app=logging-demo -o jsonpath='{.items[0].spec.containers[*].name}'
```

결과에 `datakit-logfwd`가 포함되어야 합니다. 레거시 Operator, YAML 및 logfwd 이미지는 호환되는 버전을 함께 사용해야 합니다. 업그레이드할 때는 현재 CRD 구성 방식으로 함께 마이그레이션하십시오.
