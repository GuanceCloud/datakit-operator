# DataKit Operator OpenTelemetry 주입

DataKit Operator는 [:octicons-tag-24: v1.9.0](operator-changelog.md#cl-1.9.0)부터 Java, Python 및 Node.js 애플리케이션에 OpenTelemetry 자동 계측 에이전트 주입을 지원합니다. 이 기능은 OpenTelemetry 공식 자동 주입 이미지를 사용하며 에이전트를 복사하고 시작 환경을 설정하는 공식 방식을 따릅니다.

배포 템플릿에서는 `otels`가 기본적으로 비어 있으므로 OpenTelemetry 주입은 자동으로 활성화되지 않습니다. 활성화하려면 먼저 이 페이지에 나온 규칙을 추가하십시오. 주입은 Pod 생성 시에만 수행됩니다. Operator는 Pod의 모든 일반 애플리케이션 컨테이너를 변경하지만 애플리케이션 init Container는 변경하지 않으며 컨테이너 내부의 언어 버전이나 libc도 감지하지 않습니다.

## 사전 준비 {#otel-prerequisites}

### DataKit OpenTelemetry 수집기 활성화 {#enable-datakit-otel}

DataKit에서 `opentelemetry` input을 활성화해야 합니다. 예를 들어 DataKit DaemonSet의 기본 수집기 목록에 `opentelemetry`를 추가합니다.

```yaml
- name: ENV_DEFAULT_ENABLED_INPUTS
  value: statsd,dk,cpu,ddtrace,opentelemetry
```

OTLP Trace, Metric 및 Log는 동일한 DataKit Service와 `9529` 포트를 사용하지만 요청 경로는 서로 다릅니다.

```text
Trace:  http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/traces
Metric: http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/metrics
Log:    http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/logs
```

이 페이지의 예시 규칙은 Trace만 활성화합니다. Metric 또는 Log를 수집하려면 해당 `OTEL_METRICS_EXPORTER` 또는 `OTEL_LOGS_EXPORTER`를 `none`에서 `otlp`로 변경하십시오. 새 DataKit 주소를 구성할 필요는 없습니다. 애플리케이션과 자동 계측 에이전트 자체도 해당 신호를 생성할 수 있어야 합니다.

### 언어 지원 범위 확인 {#otel-language-support}

| 언어 | 지원 범위 | 기본 이미지 |
| --- | --- | --- |
| Java | OpenTelemetry Java Agent가 지원하는 JVM | `{{.OTelJavaImage}}` |
| Python | Python 3.10~3.14, glibc Linux만 지원 | `{{.OTelPythonImage}}` |
| Node.js | Node.js 20.6 이상, glibc 및 Alpine/musl 지원 | `{{.OTelNodeJSImage}}` |

Python 2, Python 3.9 이하 및 Alpine/musl Python 이미지는 지원 범위에 포함되지 않습니다. 현재 Node.js는 일반 CommonJS 애플리케이션을 지원하며 ESM, bundler 또는 사용자 정의 loader는 지원하지 않습니다. OpenTelemetry 공식 Operator는 현재 PHP 자동 주입을 정식으로 지원하지 않습니다. 공식 PHP 자동 계측 이미지는 제공되지만 이에 대응하는 Instrumentation 구성과 표준 주입 절차는 아직 제공되지 않으므로 DataKit Operator도 현재 OTel PHP 주입을 지원하지 않습니다.

Operator는 이러한 조건을 자동으로 판단하지 않습니다. 상호 배타적인 Namespace 또는 Label Selector를 사용하여 조건에 맞는 Pod만 해당 규칙과 일치하도록 하십시오.

## Operator 구성 {#otel-config}

`otels`는 `ddtraces`와 같은 계층에 있으며 배포 템플릿의 기본값은 `[]`입니다. 다음의 전체 Java 규칙을 이 배열에 추가하면 Selector 조건과 일치하는 Pod에 대한 주입이 활성화됩니다.

```json
{
    "admission_inject_v2": {
        "otels": [
            {
                "name": "otel-java",
                "language": "java",
                "namespace_selectors": ["^production$"],
                "label_selectors": ["admission.datakit/otel-language=java"],
                "check_annotation": false,
                "image": "{{.OTelJavaImage}}",
                "envs": {
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "OTEL_SERVICE_NAME": "{fieldRef:metadata.labels['app']}",
                    "OTEL_RESOURCE_ATTRIBUTES": "k8s.pod.name=$(POD_NAME),k8s.namespace.name=$(POD_NAMESPACE),k8s.node.name=$(NODE_NAME)",
                    "OTEL_TRACES_EXPORTER": "otlp",
                    "OTEL_LOGS_EXPORTER": "none",
                    "OTEL_METRICS_EXPORTER": "none",
                    "OTEL_EXPORTER_OTLP_PROTOCOL": "http/protobuf",
                    "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/traces",
                    "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/logs",
                    "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/metrics"
                },
                "resources": {
                    "requests": {
                        "cpu": "100m",
                        "memory": "64Mi"
                    },
                    "limits": {
                        "cpu": "500m",
                        "memory": "512Mi"
                    }
                }
            }
        ]
    }
}
```

환경 변수는 구성 순서를 유지합니다. 위 예시의 `POD_NAME`, `POD_NAMESPACE` 및 `NODE_NAME`은 이를 참조하는 `OTEL_RESOURCE_ATTRIBUTES`보다 앞에 있어야 합니다.

주요 필드는 다음과 같습니다.

| 필드 | 설명 |
| --- | --- |
| `name` | 로그에서 규칙을 식별하는 이름. 구성을 권장합니다. |
| `language` | 필수. `java`, `python` 또는 `nodejs`를 사용할 수 있습니다. |
| `namespace_selectors` | Namespace 정규식 배열 |
| `label_selectors` | Pod Label Selector 배열 |
| `check_annotation` | Pod에 해당 언어의 버전 Annotation을 요구할지 여부. 기본값은 `false`입니다. |
| `image` | 필수. OpenTelemetry 공식 이미지 또는 프라이빗 레지스트리 복사본 |
| `envs` | 모든 일반 애플리케이션 컨테이너에 주입할 환경 변수 |
| `resources` | init Container 리소스 구성. 없거나 잘못된 경우 기본값을 사용합니다. |

Python과 Node.js 규칙의 필드는 동일합니다. `name`, `language`, 언어 Label 및 이미지만 변경하면 됩니다.

| 언어 | Label | 이미지 |
| --- | --- | --- |
| Python | `admission.datakit/otel-language=python` | `{{.OTelPythonImage}}` |
| Node.js | `admission.datakit/otel-language=nodejs` | `{{.OTelNodeJSImage}}` |

Selector 및 Annotation의 공통 규칙은 [DataKit Operator 주입 규칙](datakit-operator.md#datakit-operator-inject)을 참조하십시오. 하나의 Pod가 여러 OTel 규칙과 일치하면 Operator는 Annotation 조건을 충족하는 첫 번째 규칙만 사용합니다. 첫 번째 규칙의 언어 또는 이미지 구성이 잘못되어도 이후 규칙으로 대체하지 않습니다.

## 언어별 주입 방식 {#otel-injection}

모든 언어에 다음 항목이 추가됩니다.

- `datakit-otel-lib-init` init Container
- `datakit-otel-auto-instrument` EmptyDir 볼륨
- 규칙에 구성된 `OTEL_*` 등의 환경 변수

언어별 에이전트 로드 방식은 다음과 같습니다.

| 언어 | init Container 복사 방식 | 마운트 및 시작 환경 |
| --- | --- | --- |
| Java | 이미지의 `/javaagent.jar`를 공유 볼륨으로 복사 | `/otel-auto-instrumentation-java`를 마운트하고 `JAVA_TOOL_OPTIONS`에 `-javaagent:/otel-auto-instrumentation-java/javaagent.jar` 추가 |
| Python | 이미지의 `/autoinstrumentation/`을 공유 볼륨으로 복사 | `/otel-auto-instrumentation-python`을 마운트하고 `PYTHONPATH`의 앞뒤에 자동 초기화 디렉터리와 에이전트 디렉터리를 추가한 후 `sitecustomize.py`로 로드 |
| Node.js | 이미지의 `/autoinstrumentation/`을 공유 볼륨으로 복사 | `/otel-auto-instrumentation-nodejs`를 마운트하고 `NODE_OPTIONS`에 `--require /otel-auto-instrumentation-nodejs/autoinstrumentation.js` 추가 |

Operator는 기존 일반 문자열 값을 유지합니다. 해당 시작 환경이 `valueFrom`을 사용하거나, 중복 항목이 있거나, Datadog 에이전트가 이미 로드되어 있거나, 기존 OTel init Container, 볼륨 또는 마운트가 예상 구성과 충돌하면 전체 OTel 주입을 건너뛰고 warning을 기록합니다. Admission은 계속 fail-open 방식으로 동작하므로 애플리케이션 Pod 생성을 차단하지 않습니다.

## Annotation 및 버전 {#otel-annotations}

개별 Pod에서 OTel을 비활성화하려면 `admission.datakit/otel.enabled: "false"`를 설정합니다. 이 설정은 `check_annotation`과 관계없이 항상 적용됩니다.

규칙에서 `check_annotation: true`를 설정하면 Pod에 해당 언어의 버전 Annotation도 있어야 합니다.

| 언어 | 버전 Annotation |
| --- | --- |
| Java | `admission.datakit/otel-java-lib.version` |
| Python | `admission.datakit/otel-python-lib.version` |
| Node.js | `admission.datakit/otel-nodejs-lib.version` |

버전 값은 규칙 `image`의 태그만 대체하며 이미지 레지스트리나 이름은 변경하지 않습니다. 플랫폼에서 버전을 중앙 관리한다면 `check_annotation: false`를 유지하는 것이 좋습니다.

## DDTrace와의 관계 {#otel-ddtrace-conflict}

하나의 컨테이너에 DDTrace와 OpenTelemetry 자동 계측 에이전트를 동시에 로드하면 안 됩니다. DDTrace와 OTel 규칙이 동시에 일치하면 DDTrace가 우선합니다. DDTrace가 선택된 후 주입에 실패해도 OTel로 대체하지 않습니다.

OTel 워크로드에서는 DDTrace를 명시적으로 비활성화하는 것이 좋습니다.

```yaml
admission.datakit/ddtrace.enabled: "false"
```

## `runAsNonRoot` 보호 {#otel-run-as-non-root}

OpenTelemetry 공식 자동 주입 이미지의 기본 사용자는 애플리케이션 Pod의 보안 정책과 일치하지 않을 수 있습니다. Operator는 첫 번째 애플리케이션 컨테이너의 SecurityContext를 그대로 사용하여 OTel init Container를 생성합니다.

init Container의 유효 구성에서 `runAsNonRoot: true`이지만 0이 아닌 `runAsUser`를 명시하지 않으면 Kubernetes가 이미지의 비 root 사용자 실행 여부를 확인할 수 없어 시작을 거부할 수 있습니다. 주입 때문에 애플리케이션 Pod가 초기화 단계에서 멈추는 것을 방지하기 위해 Operator는 전체 OTel 주입을 건너뛰고 다음 원인이 포함된 warning을 기록합니다.

```text
reason=run_as_non_root_without_run_as_user
```

이러한 Pod에서 OTel을 활성화하려면 첫 번째 애플리케이션 컨테이너 또는 Pod에 사용 중인 OTel 이미지와 호환되는 0이 아닌 `runAsUser`를 명시하십시오. `runAsNonRoot: true`인 경우 프라이빗 이미지가 실제로 비 root 사용자를 사용하더라도 `runAsUser`가 명시되지 않으면 안전을 위해 주입을 건너뜁니다.

## Deployment 예시 {#otel-example}

다음 Java Deployment는 앞의 구성과 일치합니다.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: order-service
  namespace: production
spec:
  replicas: 1
  selector:
    matchLabels:
      app: order-service
  template:
    metadata:
      labels:
        app: order-service
        admission.datakit/otel-language: java
      annotations:
        admission.datakit/ddtrace.enabled: "false"
        admission.datakit/otel.enabled: "true"
    spec:
      containers:
        - name: app
          image: example/order-service:1.0.0
          ports:
            - name: http
              containerPort: 8080
```

Label 값을 `python` 또는 `nodejs`로 변경하면 해당 언어 규칙과 일치합니다. 애플리케이션 이미지는 앞서 설명한 지원 범위를 충족해야 합니다.

## 검증 및 문제 해결 {#otel-verify}

Pod를 생성한 후 주입 결과를 확인합니다.

```shell
kubectl -n production get pod -l app=order-service -o yaml
kubectl logs -n datakit deployment/datakit-operator
```

최종 Pod에는 `datakit-otel-lib-init`, `datakit-otel-auto-instrument`, 해당 언어의 마운트, 시작 환경 및 `OTEL_*` 환경 변수가 있어야 합니다. 그런 다음 애플리케이션에 실제 요청을 보내고 페이지에서 다음 조건으로 조회합니다.

```text
service:order-service
source:opentelemetry
```

일반적인 문제는 다음과 같습니다.

- 실행 중인 Pod가 변경되지 않음: Pod를 다시 생성하십시오. Operator는 `CREATE`만 처리합니다.
- 주입되지 않음: Namespace, Label, `check_annotation`, DDTrace 우선순위 및 Operator warning을 확인하십시오.
- init Container 이미지 가져오기 실패: 공식 이미지는 `imagePullPolicy: Always`를 사용합니다. GHCR 네트워크를 확인하거나 이미지를 프라이빗 레지스트리에 동기화한 후 규칙을 변경하십시오.
- 주입되었지만 데이터가 없음: DataKit에서 `opentelemetry` input을 활성화했는지, OTLP 주소에 접근할 수 있는지, 애플리케이션 프레임워크가 자동 계측 에이전트에서 지원되는지 확인하고 실제 요청을 한 번 보내십시오.
- 롤백 필요: OTel 규칙을 삭제하거나 비활성화한 후 이미 주입된 Pod를 다시 생성하십시오.
