# DataKit Operator Flameshot 주입

DataKit Operator는 [:octicons-tag-24: v1.8.0](operator-changelog.md#cl-1.8.0)부터 Flameshot 사이드카 주입을 지원합니다. Flameshot은 일정 또는 리소스 임계값에 따라 Java, Python 및 Go 애플리케이션의 Profiling 데이터를 수집하며 레거시 Profiler 주입을 대체합니다.

## 사전 요구 사항 {#flameshot-prerequisites}

- 클러스터에 [DataKit](https://docs.<<<custom_key.brand_main_domain>>>/datakit/datakit-daemonset-deploy/){:target="_blank"}이 설치되어 있어야 합니다.
- DataKit에서 [Profile 수집기](https://docs.<<<custom_key.brand_main_domain>>>/datakit/datakit-daemonset-deploy/#using-k8-env){:target="_blank"}가 활성화되어 있어야 합니다.
- 대상 Pod의 보안 정책에서 사이드카에 `SYS_PTRACE` capability를 추가할 수 있어야 합니다.
- Prometheus Annotation을 활성화하려면 DataKit에서 KubernetesPrometheus를 활성화하고 Pod Annotation 자동 검색도 활성화해야 합니다.

## Operator 구성 {#flameshot-usage}

`admission_inject_v2.flameshots`에 규칙을 추가합니다.

```json
{
    "admission_inject_v2": {
        "flameshots": [
            {
                "name": "flameshot-java",
                "namespace_selectors": ["^production$"],
                "label_selectors": ["profiling=flameshot"],
                "image": "{{.FlameshotImage}}",
                "envs": {
                    "FLAMESHOT_DATAKIT_ADDR": "http://datakit-service.datakit:9529/profiling/v1/input",
                    "FLAMESHOT_MONITOR_INTERVAL": "10s",
                    "FLAMESHOT_LOG_LEVEL": "info",
                    "FLAMESHOT_PROFILING_PATH": "/flameshot-data",
                    "FLAMESHOT_LOG_PATH": "/var/log/flameshot.log",
                    "FLAMESHOT_HTTP_LOCAL_IP": "{fieldRef:status.podIP}",
                    "FLAMESHOT_HTTP_LOCAL_PORT": "8089"
                },
                "processes": "[{\"service\":\"java-demo\",\"language\":\"java\",\"command\":\"^java\\\\b.*app\\\\.jar$\",\"events\":\"cpu\",\"duration\":\"30s\",\"cpu_usage_percent\":80}]",
                "enable_prometheus_annotations": true,
                "resources": {
                    "requests": {
                        "cpu": "100m",
                        "memory": "128Mi"
                    },
                    "limits": {
                        "cpu": "200m",
                        "memory": "256Mi"
                    }
                }
            }
        ]
    }
}
```

주요 필드는 다음과 같습니다.

| 필드 | 설명 |
| --- | --- |
| `name` | 로그에서 규칙을 식별하는 이름. 구성을 권장합니다. |
| `namespace_selectors` | Namespace 정규식 배열 |
| `label_selectors` | Pod Label Selector 배열 |
| `image` | Flameshot 사이드카 이미지 |
| `envs` | 사이드카 환경 변수 |
| `processes` | 필수이며 비워 둘 수 없습니다. 프로세스 일치 및 수집 정책을 정의하는 JSON 배열 문자열입니다. |
| `enable_prometheus_annotations` | Flameshot 메트릭 수집 Annotation을 자동으로 추가할지 여부. 기본값은 `false`입니다. |
| `resources` | 사이드카 리소스 구성. 없거나 잘못된 경우 기본값을 사용합니다. |

주입하려면 `FLAMESHOT_PROFILING_PATH`와 유효한 `FLAMESHOT_HTTP_LOCAL_PORT`가 필요합니다. 둘 중 하나라도 없거나 `processes`가 비어 있으면 Operator는 주입을 건너뛰고 warning을 기록합니다.

Selector 및 Annotation의 공통 규칙은 [DataKit Operator 주입 규칙](datakit-operator.md#datakit-operator-inject)을 참조하십시오. 개별 Pod에서 Flameshot을 비활성화하려면 `admission.datakit/flameshot.enabled: "false"`를 설정합니다.

## 주입 결과 {#flameshot-injection-result}

규칙이 일치하면 Operator는 다음 작업을 수행합니다.

- `datakit-flameshot` 사이드카를 추가하고 `SYS_PTRACE` capability를 부여합니다.
- 사이드카에서 애플리케이션 프로세스를 검색할 수 있도록 Pod에 공유 프로세스 네임스페이스를 설정합니다.
- `flameshot-volume` EmptyDir를 생성하고 모든 일반 컨테이너의 `FLAMESHOT_PROFILING_PATH`에 마운트합니다.
- `processes`를 `FLAMESHOT_PROCESSES`로 사이드카에 주입합니다.
- Pod의 `restartPolicy`를 `Always`로 설정합니다.

Flameshot은 애플리케이션 프로세스에 직접 접근합니다. 운영 환경에 적용하기 전에 Pod Security Admission, 컨테이너 보안 정책 및 애플리케이션 실행 환경에서 위 변경 사항을 허용하는지 확인하십시오.

## 수집 구성 {#envs}

주요 환경 변수는 다음과 같습니다.

| 환경 변수 | 설명 |
| --- | --- |
| `FLAMESHOT_DATAKIT_ADDR` | DataKit Profiling 수신 주소 |
| `FLAMESHOT_MONITOR_INTERVAL` | 프로세스 및 리소스 모니터링 간격 |
| `FLAMESHOT_LOG_LEVEL` | Flameshot 로그 수준 |
| `FLAMESHOT_PROFILING_PATH` | Profiling 임시 파일 공유 디렉터리. 주입에 필요합니다. |
| `FLAMESHOT_LOG_PATH` | Flameshot 로그 경로 |
| `FLAMESHOT_HTTP_LOCAL_IP` | Flameshot HTTP 수신 IP |
| `FLAMESHOT_HTTP_LOCAL_PORT` | Flameshot HTTP 및 메트릭 포트. 주입에 필요합니다. |
| `FLAMESHOT_SERVICE` | 모든 프로세스 규칙의 service 덮어쓰기 |
| `FLAMESHOT_TAGS` | 전역 Profiling 태그 |
| `FLAMESHOT_POD_CPU_LIMIT` | Pod CPU limit. 단위는 millicore입니다. |
| `FLAMESHOT_POD_MEM_LIMIT` | Pod 메모리 limit. 단위는 MiB입니다. |

`processes`는 Java, Python 및 Go의 명령 일치, 수집 시간, CPU/메모리 임계값 및 언어별 옵션을 지원합니다. Heap Dump와 오브젝트 스토리지 업로드도 기존 `envs`로 구성합니다. 민감한 자격 증명에는 `{secretKeyRef:<SECRET>.<KEY>}` 사용을 권장합니다. 전체 필드는 [Flameshot 문서](../integrations/flameshot.md)를 참조하십시오.

### Prometheus Annotation {#prom-anno}

`enable_prometheus_annotations: true`이면 Operator가 다음 항목을 추가합니다.

```yaml
prometheus.io/scrape: "true"
prometheus.io/port: "8089"
prometheus.io/scheme: "http"
prometheus.io/path: "/metrics"
prometheus.io/param_measurement: "flameshot"
```

포트는 `FLAMESHOT_HTTP_LOCAL_PORT`에서 가져옵니다. Pod에 `prometheus.io/`로 시작하는 Annotation이 하나라도 있으면 Operator는 사용자 구성을 유지하고 위 Annotation을 추가하지 않습니다.

## Deployment 예시 {#flameshot-example}

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: java-demo
  namespace: production
spec:
  replicas: 1
  selector:
    matchLabels:
      app: java-demo
  template:
    metadata:
      labels:
        app: java-demo
        profiling: flameshot
      annotations:
        admission.datakit/flameshot.enabled: "true"
    spec:
      containers:
        - name: app
          image: example/java-demo:1.0.0
```

생성한 후 다음을 확인합니다.

```shell
kubectl -n production get pod -l app=java-demo -o jsonpath='{.items[0].spec.containers[*].name}'
kubectl -n production logs -l app=java-demo -c datakit-flameshot
```

결과에 `datakit-flameshot`이 포함되어야 합니다. Profiling 데이터가 생성되면 <<<custom_key.brand_name>>>의 Profiling 페이지에서 확인할 수 있습니다. 데이터가 없으면 먼저 사이드카 로그, DataKit 주소, `processes`의 명령 정규식 및 대상 프로세스 권한을 확인하십시오.
