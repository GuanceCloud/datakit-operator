# DataKit Operator DDTrace 주입

DataKit Operator는 Pod 생성 시 DDTrace 자동 계측 에이전트를 주입하며 Java, Python, PHP 및 Node.js를 지원합니다. Operator는 Pod에 `datakit-lib-init` init Container와 `/datadog-lib` 공유 볼륨을 추가하고 모든 일반 애플리케이션 컨테이너의 시작 환경을 변경합니다. 기존 Pod는 변경되지 않으므로 다시 생성해야 적용됩니다.

## 사용 방법 {#datakit-operator-inject-lib-usage}

먼저 [DataKit Operator를 설치](datakit-operator.md#install)하고 애플리케이션 컨테이너에서 DataKit Trace 수신 주소에 접근할 수 있는지 확인하십시오.

배포 템플릿은 기본적으로 Java DDTrace 규칙 하나만 유지하며 `default` Namespace의 모든 Pod와 일치시킵니다. 이는 기존 배포와의 호환성을 유지하기 위한 것이며 Java만 지원한다는 의미는 아닙니다. Operator는 애플리케이션 컨테이너의 언어를 자동으로 감지하지 않습니다. Python, PHP 또는 Node.js를 구성할 때는 기본 Java 규칙도 상호 배타적인 언어 Label을 사용하도록 변경해야 합니다. 그렇지 않으면 해당 규칙이 먼저 Pod와 일치하여 뒤에 있는 언어 규칙이 적용되지 않습니다.

### 지원 범위 및 이미지 {#ddtrace-lib-image-selection}

| 언어 | 애플리케이션 런타임 | 기본 이미지 |
| --- | --- | --- |
| Java | DDTrace Java Agent가 지원하는 JVM | `{{.DDTraceJavaImage}}` |
| Python | Python 3.7 | `{{.DDTracePython37Image}}` |
| Python | Python 3.8 | `{{.DDTracePython38Image}}` |
| Python | Python 3.9~3.14 | `{{.DDTracePythonImage}}` |
| PHP | Linux GNU 또는 musl | `{{.DDTracePHPImage}}` |
| Node.js | Node.js 16 | `{{.DDTraceNodeJS16Image}}` |
| Node.js | Node.js 18~25 | `{{.DDTraceNodeJSImage}}` |

이미지 버전은 애플리케이션 컨테이너의 언어 런타임과 호환되어야 합니다. 오프라인 환경에서는 이미지를 프라이빗 레지스트리에 동기화하고 규칙의 `image`에 전체 주소를 입력할 수 있습니다.

PHP 규칙에는 `php_loader_flavor`도 구성해야 합니다. glibc 이미지는 `linux-gnu`, Alpine 등의 musl 이미지는 `linux-musl`을 사용합니다. Operator는 애플리케이션 이미지의 libc를 자동으로 감지하지 않습니다. 구성하지 않았거나 잘못 구성하면 `linux-gnu`로 대체합니다.

PHP init Container는 해당 libc의 loader 구성을 복사합니다. 애플리케이션 프로세스가 시작되면 Datadog loader가 PHP 버전, ABI 및 ZTS/NTS 모드에 따라 호환되는 `.so` 파일을 로드합니다. Operator 자체는 libc 유형만 선택하며 PHP 런타임은 검사하지 않습니다.

### 구성 규칙 {#ddtrace-config}

다음 구성에는 Java, Python, PHP 및 Node.js 규칙이 모두 포함되어 있습니다. 기존 `jsonconfig`를 편집할 때는 이 `ddtraces` 배열 전체로 `admission_inject_v2.ddtraces`를 교체하고 `admission_inject_v2` 아래의 다른 구성은 유지하십시오. 배포 템플릿의 기본 Java 규칙 뒤에 이 규칙들을 그대로 추가하지 마십시오. 기본 규칙이 먼저 일치합니다.

모든 규칙은 `"namespace_selectors": ["*"]`를 사용하지만 해당 언어 Label이 있는 Pod만 일치합니다. 따라서 Label이 없는 Pod에 자동으로 주입하지 않으면서 여러 Namespace에서 사용할 수 있습니다. 예시의 Python 이미지는 Python 3.9~3.14, PHP는 `linux-gnu`, Node.js 이미지는 Node.js 18~25용입니다. 다른 런타임에서는 앞의 표에 따라 이미지 또는 `php_loader_flavor`를 변경하십시오.

```json
{
    "admission_inject_v2": {
        "ddtraces": [
            {
                "name": "ddtrace-java",
                "language": "java",
                "namespace_selectors": ["*"],
                "label_selectors": ["admission.datakit/ddtrace-language=java"],
                "check_annotation": false,
                "image": "{{.DDTraceJavaImage}}",
                "envs": {
                    "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DD_TRACE_AGENT_PORT": "9529",
                    "DD_JMXFETCH_STATSD_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DD_JMXFETCH_STATSD_PORT": "8125",
                    "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
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
            },
            {
                "name": "ddtrace-python",
                "language": "python",
                "namespace_selectors": ["*"],
                "label_selectors": ["admission.datakit/ddtrace-language=python"],
                "check_annotation": false,
                "image": "{{.DDTracePythonImage}}",
                "envs": {
                    "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DD_TRACE_AGENT_PORT": "9529",
                    "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
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
            },
            {
                "name": "ddtrace-php",
                "language": "php",
                "php_loader_flavor": "linux-gnu",
                "namespace_selectors": ["*"],
                "label_selectors": ["admission.datakit/ddtrace-language=php"],
                "check_annotation": false,
                "image": "{{.DDTracePHPImage}}",
                "envs": {
                    "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DD_TRACE_AGENT_PORT": "9529",
                    "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
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
            },
            {
                "name": "ddtrace-nodejs",
                "language": "nodejs",
                "namespace_selectors": ["*"],
                "label_selectors": ["admission.datakit/ddtrace-language=nodejs"],
                "check_annotation": false,
                "image": "{{.DDTraceNodeJSImage}}",
                "envs": {
                    "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DD_TRACE_AGENT_PORT": "9529",
                    "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
                    "POD_NAME": "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
                    "NODE_NAME": "{fieldRef:spec.nodeName}",
                    "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
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

Pod에는 해당 언어 Label을 하나만 설정하십시오. 예:

```yaml
metadata:
  labels:
    app: payment-service
    admission.datakit/ddtrace-language: python
```

주요 필드는 다음과 같습니다.

| 필드 | 설명 |
| --- | --- |
| `name` | 로그에서 규칙을 식별하는 이름. 구성을 권장합니다. |
| `language` | 필수. `java`, `python`, `php` 또는 `nodejs`를 사용할 수 있습니다. |
| `namespace_selectors` | Namespace 정규식 배열 |
| `label_selectors` | Pod Label Selector 배열 |
| `check_annotation` | Pod에 해당 언어의 버전 Annotation을 요구할지 여부. 기본값은 `false`입니다. |
| `image` | 필수. 언어 라이브러리 init Container 이미지 |
| `envs` | 모든 일반 애플리케이션 컨테이너에 주입할 환경 변수 |
| `resources` | init Container 리소스 구성. 없거나 잘못된 경우 기본값을 사용합니다. |
| `php_loader_flavor` | PHP에서만 사용. `linux-gnu` 또는 `linux-musl`을 선택할 수 있습니다. |

Selector, Annotation, 기본 리소스 및 환경 변수 참조의 공통 규칙은 [DataKit Operator 주입 규칙](datakit-operator.md#datakit-operator-inject)을 참조하십시오. 하나의 Pod가 여러 규칙에 일치하지 않도록 언어별 규칙에 상호 배타적인 Label을 사용하십시오.

## 주입 방식 {#ddtrace-injection}

| 언어 | Operator가 애플리케이션 컨테이너에 적용하는 변경 |
| --- | --- |
| Java | `/datadog-lib`를 마운트하고 `JAVA_TOOL_OPTIONS`에 `-javaagent:/datadog-lib/dd-java-agent.jar` 추가 |
| Python | `/datadog-lib`를 마운트하고 `/datadog-lib/`를 `PYTHONPATH` 앞에 추가 |
| PHP | `/datadog-lib`를 마운트하고 `DD_LOADER_PACKAGE_PATH` 및 `PHP_INI_SCAN_DIR`를 구성하여 `dd_library_loader.ini` 로드 |
| Node.js | `/datadog-lib`를 마운트하고 `NODE_OPTIONS`에 `--require=/datadog-lib/node_modules/dd-trace/init` 추가 |

Operator는 애플리케이션 컨테이너에 이미 있는 동일한 이름의 시작 인수를 유지합니다. `JAVA_TOOL_OPTIONS`, `PYTHONPATH`, `PHP_INI_SCAN_DIR` 또는 `NODE_OPTIONS`가 Kubernetes `valueFrom`을 사용하면 문자열을 안전하게 병합할 수 없으므로 전체 DDTrace 주입을 건너뛰고 warning을 기록합니다. 따라서 동작하지 않는 init Container만 남는 것을 방지합니다.

규칙의 일반 환경 변수는 애플리케이션 컨테이너에 있는 동일한 이름의 변수를 덮어쓰지 않습니다. `DD_TAGS`는 예외입니다. 양쪽 값이 모두 일반 문자열이면 Operator가 태그를 병합합니다.

## Annotation 및 버전 {#check-annotation-config}

개별 Pod에서 DDTrace를 비활성화하려면 `admission.datakit/ddtrace.enabled: "false"`를 설정합니다. 이 설정은 `check_annotation`과 관계없이 항상 적용됩니다.

규칙에서 `check_annotation: true`를 설정하면 Pod에 해당 언어의 버전 Annotation도 있어야 합니다.

| 언어 | 버전 Annotation |
| --- | --- |
| Java | `admission.datakit/java-lib.version` |
| Python | `admission.datakit/python-lib.version` |
| PHP | `admission.datakit/php-lib.version` |
| Node.js | `admission.datakit/nodejs-lib.version` |

버전 값은 규칙 `image`의 태그만 대체하며 이미지 레지스트리나 이름은 변경하지 않습니다. 따라서 동일한 이미지의 버전을 전환할 때만 사용할 수 있습니다.

여러 DDTrace 규칙이 동시에 일치하면 Operator는 구성 순서에 따라 Annotation 조건을 충족하는 첫 번째 규칙을 사용합니다. DDTrace와 OpenTelemetry가 동시에 일치하면 DDTrace가 우선합니다. DDTrace가 선택된 후 주입에 실패해도 OpenTelemetry로 대체하지 않습니다. 동일한 Pod에 두 자동 계측 에이전트를 동시에 활성화하지 마십시오.

## Deployment 예시 {#anno-demo}

다음 Deployment는 앞의 Java 규칙과 일치합니다.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: java-demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: java-demo
  template:
    metadata:
      labels:
        app: java-demo
        admission.datakit/ddtrace-language: java
      annotations:
        admission.datakit/ddtrace.enabled: "true"
    spec:
      containers:
        - name: app
          image: example/java-demo:1.0.0
```

DDTrace를 명시적으로 비활성화하려면 Annotation을 다음과 같이 변경합니다.

```yaml
admission.datakit/ddtrace.enabled: "false"
```

## 검증 및 문제 해결 {#ddtrace-verify}

Pod를 다시 생성한 후 최종 Pod를 확인합니다.

```shell
kubectl get pod <pod-name> -o jsonpath='{.spec.initContainers[*].name}'
kubectl get pod <pod-name> -o yaml
kubectl logs -n datakit deployment/datakit-operator
```

`datakit-lib-init`, `datakit-auto-instrument` 볼륨, 해당 언어의 마운트 및 시작 환경이 있어야 합니다. 그런 다음 실제 요청을 한 번 보내고 DataKit monitor 또는 <<<custom_key.brand_name>>> 페이지에서 Trace 데이터를 확인합니다.

일반적인 문제는 다음과 같습니다.

- 실행 중인 Pod가 변경되지 않음: Pod를 다시 생성하십시오. Operator는 `CREATE`만 처리합니다.
- 주입되지 않음: Namespace, Label 및 `check_annotation` 조건을 모두 충족하는지 확인하십시오.
- init Container 이미지 가져오기 실패: 규칙의 이미지 주소, 자격 증명 및 클러스터 네트워크를 확인하십시오.
- init Container는 성공했지만 Trace가 없음: 애플리케이션 프로세스에 주입된 시작 환경이 유지되는지 확인하고 `DD_AGENT_HOST` 및 `DD_TRACE_AGENT_PORT`에 접근할 수 있는지 확인하십시오.
- PHP 시작 실패: 애플리케이션 이미지가 glibc와 musl 중 무엇을 사용하는지 확인하고 올바른 `php_loader_flavor`를 설정하십시오.
