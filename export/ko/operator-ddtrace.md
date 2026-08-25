# DataKit Operator의 DDTrace 주입

DataKit Operator는 Pod **생성 시** admission webhook을 통해 Pod 템플릿을 변경합니다. `datakit-lib-init` initContainer를 추가하고 언어 라이브러리를 공유 볼륨 `/datadog-lib`에 복사한 다음, 해당 볼륨과 필수 시작 환경을 애플리케이션 컨테이너에 주입합니다. 이미 실행 중인 Pod는 변경하지 않습니다. Operator ConfigMap, 이미지 버전 또는 Deployment 어노테이션을 변경한 후에는 배포하거나 재시작하여 새 Pod를 생성해야 변경 사항이 적용됩니다.

사용하기 전에 다음 세 가지를 확인해야 합니다.

1. 선택기는 주입할 워크로드를 결정합니다. 빈 선택기는 영향 범위를 확대할 수 있으므로 먼저 별도의 네임스페이스에서 시험해야 합니다.
1. `image`의 라이브러리 버전은 initContainer의 런타임이 아니라 **애플리케이션 컨테이너**의 언어 런타임 버전과 일치해야 합니다.
1. 라이브러리를 주입했다고 해서 대상 애플리케이션이 반드시 시작되거나 트레이스를 생성하는 것은 아닙니다. 애플리케이션 컨테이너의 시작 인수, 환경 변수 및 실제 trace 요청을 확인해야 합니다.

## 사용 안내 {#datakit-operator-inject-lib-usage}

1. 대상 Kubernetes 클러스터에서 [DataKit-Operator를 다운로드하고 설치합니다](datakit-operator.md#install).
1. Operator에 다음 ConfigMap 설정을 추가합니다.

    ```json
    {
        "server_listen": "0.0.0.0:9543",
        "log_level": "info",
        "admission_inject_v2": {
            "ddtraces": [
                {
                    "namespace_selectors": ["staging"],
                    "label_selectors": ["app=example"],
                    "check_annotation": false,
                    "image": "<ddtrace-library-image>",
                    "language": "java",
                    "envs": {
                        "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                        "DD_TRACE_AGENT_PORT": "9529"
                    }
                }
            ]
        },
        "admission_inject": {
            "ddtrace": {}
        }
    }
    ```

    위 예시는 파싱 가능한 JSON입니다. `admission_inject_v2`(Operator `v1.8.0+`)은 여러 DDTrace 설정을 지원하므로 우선 사용하는 것이 좋습니다. `admission_inject`은 이전 버전과의 호환을 위한 설정이며 일반적으로 하나의 DDTrace 규칙만 표현할 수 있습니다. `//` 주석이 포함된 JSON을 ConfigMap에 직접 복사하지 마십시오.

    DDTrace 주입에는 다음 설정 필드가 있습니다.

    | 필드                         | 유형    | 설명                                                   | 필수 여부     | 예시 값                           |
    | ------:                      | :-----: | :------                                                | :---:        | :--------                        |
    | `envs`                       | object  | 환경 변수 매핑                                           | Y[^envs]     | 아래 예시 참조                       |
    | `image`                      | string  | DDTrace 이미지 주소                                       | Y[^image]    | 아래 예시 참조                       |
    | `label_selectors`            | array   | 레이블 선택기 배열                                         | Y[^selector] | `["app=nginx", "tier=frontend"]` |
    | `language`                   | string  | 지원 언어 유형(선택 가능: `java`/`python`/`php`/`nodejs`)   | Y[^lang]     | `"nodejs"`                       |
    | `namespace_selectors`        | array   | 정규 표현식을 사용하는 네임스페이스 선택기                         | Y[^selector] | `["^prod-.*$", "^test$"]`       |
    | `resources`                  | object  | 리소스 제한 설정                                           | N            | 아래 예시 참조                       |
    | ~~`enabled_namespaces`~~     | object  | 주입할 Kubernetes namespace를 선택하고 해당 개발 언어를 설정 | Y            | 1.7.0에서 `admission_inject_v2`은 더 이상 사용되지 않음|
    | ~~`enabled_labelselectors`~~ | object  | Kubernetes label을 통해 주입 대상을 선택                 | Y            | 1.7.0에서 `admission_inject_v2`은 더 이상 사용되지 않음|

    [^selector]: 필드 자체를 반드시 입력해야 하며, 그렇지 않으면 Operator가 주입을 거부합니다. 빈 배열은 선택 범위를 넓히므로 운영에 배포하기 전에 예상한 범위인지 확인해야 합니다.
    [^image]: 설치 템플릿에서 기본 이미지 주소를 제공합니다. 오프라인 환경에서는 일반적으로 이미지를 내부 네트워크로 복사하고 내부 이미지 주소를 사용해야 합니다.
    [^lang]: 여기에서 선택한 언어는 해당 DDTrace 이미지의 콘텐츠와 일치해야 합니다. 일치하지 않으면 주입이 실패합니다.
    [^envs]: 이러한 환경 변수 설정은 매우 중요하며 최종 데이터 결과에 직접 영향을 줍니다. 지원되는 `fieldRef` 목록은 [여기](datakit-operator.md#downwardapi)를 참조하십시오.

    `language` 필드에서 허용되는 값이 모든 버전에 사용 가능한 이미지가 존재함을 의미하지는 않습니다. 이 페이지에서는 검증된 Node.js 및 Python 버전 매핑을 제공합니다. Java는 아래 예시를 사용하고 최종 JVM 명령이 Agent를 로드했는지 확인해야 합니다. PHP는 설치 템플릿이나 버전 정보에서 PHP 실행 방식과 명확히 일치하는 이미지만 사용해야 합니다. 한 언어의 이미지를 다른 런타임에 주입하지 마십시오.

### 소규모 우선 시험 {#pilot}

먼저 테스트 namespace 하나와 명확한 `app=<name>` 레이블 선택기를 사용한 다음 점진적으로 범위를 확대하는 것이 좋습니다. 이미지 태그는 구체적인 버전으로 고정하고 운영 규칙에서는 `latest`를 사용하지 마십시오. 변경할 때마다 최소한 다음을 확인해야 합니다.

```shell
kubectl get pod <pod-name> -o jsonpath='{.spec.initContainers[*].name}'
kubectl describe pod <pod-name>
```

첫 번째 명령의 결과에는 `datakit-lib-init`가 포함되어야 합니다. 두 번째 명령은 마운트, 환경 변수 및 webhook 이벤트를 확인하는 데 사용합니다. 이후 애플리케이션 컨테이너에서 언어 시작 인수를 확인하고 DataKit monitor에서 trace 요청도 확인해야 합니다.

### DDTrace Lib 주입 방식 및 이미지 선택 {#ddtrace-lib-image-selection}

DataKit Operator는 DDTrace를 주입할 때 `datakit-lib-init`이라는 initContainer를 추가하여 DDTrace 라이브러리를 공유 볼륨 `/datadog-lib`에 복사한 다음 해당 디렉터리를 애플리케이션 컨테이너에 마운트합니다. 이미지 버전은 initContainer의 런타임이 아니라 **애플리케이션 컨테이너 내부의 언어 런타임 버전**을 기준으로 선택해야 합니다.

이미지 저장소는 브랜드별로 구분됩니다. 이 문서의 예시에서는 `pubrepo.<<<custom_key.brand_main_domain>>>/datakit-operator`를 일관되게 사용하며, 다른 브랜드 사이트에 게시할 때 해당 저장소 주소로 교체됩니다.

#### Node.js 주입 {#ddtrace-nodejs-injection}

Node.js 주입은 애플리케이션 컨테이너에 `NODE_OPTIONS`를 설정하거나 추가합니다.

```shell
--require=/datadog-lib/node_modules/dd-trace/init
```

애플리케이션 컨테이너에 `NODE_OPTIONS`가 이미 있으면 Operator는 기존 값 뒤에 위 인수를 추가합니다. Node.js 이미지는 애플리케이션 컨테이너의 Node.js 주 버전과 일치해야 합니다.

애플리케이션이 `NODE_OPTIONS`를 직접 덮어쓰면 주입된 인수가 유실되어 trace가 생성되지 않습니다. 배포 후 `kubectl exec`를 통해 `NODE_OPTIONS`를 확인하고 `--require=/datadog-lib/node_modules/dd-trace/init`가 유지되는지 확인할 수 있습니다.

| 애플리케이션 컨테이너의 Node.js 버전 | 권장 이미지 | 버전 요구 사항 |
| --- | --- | --- |
| Node.js 18~25 | `{{.DDTraceNodeJSImage}}` | 기본 권장 버전이며 `dd-trace@5.102.0`가 내장되어 있고 `node >=18 <26`가 필요함 |
| Node.js 16 | `{{.DDTraceNodeJS16Image}}` | `dd-trace` 4.x 계열 사용. Node.js 16에는 `5.102.0` 사용을 권장하지 않음 |
| Node.js 14 이하 | 기본 지원 범위에 포함되지 않음 | 더 이전의 `dd-trace` 주 버전과 해당 이미지가 필요함. Node.js를 우선 업그레이드하는 것이 좋음 |

Node.js DDTrace 설정 예시:

```json
{
    "namespace_selectors": ["default"],
    "label_selectors": [],
    "check_annotation": false,
    "image": "{{.DDTraceNodeJSImage}}",
    "language": "nodejs",
    "envs": {
        "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
        "DD_TRACE_AGENT_PORT": "9529",
        "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
        "POD_NAME": "{fieldRef:metadata.name}",
        "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
        "NODE_NAME": "{fieldRef:spec.nodeName}",
        "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
    }
}
```

#### Python 주입 {#ddtrace-python-injection}

Python 주입은 애플리케이션 컨테이너에 `PYTHONPATH=/datadog-lib/`를 설정하거나 추가하여 Python 프로세스가 `/datadog-lib`에서 DDTrace 관련 라이브러리와 주입 bootstrap을 로드하도록 합니다. Python용 DDTrace에는 CPython ABI 관련 wheel이 포함되므로 이미지 버전이 애플리케이션 컨테이너의 Python 부 버전과 일치해야 합니다. 버전이 일치하지 않으면 `ModuleNotFoundError`, native extension 로드 실패 또는 시작 실패가 발생할 수 있습니다.

애플리케이션 이미지나 시작 스크립트에서 `PYTHONPATH`를 직접 덮어쓰는 경우 `/datadog-lib/`를 유지해야 합니다. 그렇지 않으면 주입된 라이브러리를 로드할 수 없습니다. 시작 후 `python -c 'import ddtrace; print(ddtrace.__version__)'`를 실행하여 최소 로드 검증을 수행할 수 있습니다.

| 애플리케이션 컨테이너의 Python 버전 | 권장 이미지 | 버전 요구 사항 |
| --- | --- | --- |
| Python 3.7 | `{{.DDTracePython37Image}}` | `ddtrace` 2.x는 Python 3.7을 지원함. Python 3.7은 `ddtrace` 3.x/4.x를 지원하지 않음 |
| Python 3.8 | `{{.DDTracePython38Image}}` | `ddtrace` 3.x는 Python 3.8을 지원함. `ddtrace` 4.x에는 Python 3.9 이상이 필요함 |
| Python 3.9~3.14 | `{{.DDTracePythonImage}}` | `ddtrace` 4.x에는 현재 `python >=3.9,<3.15`가 필요함 |

Python DDTrace 설정 예시:

```json
{
    "namespace_selectors": ["default"],
    "label_selectors": [],
    "check_annotation": false,
    "image": "{{.DDTracePythonImage}}",
    "language": "python",
    "envs": {
        "DD_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
        "DD_TRACE_AGENT_PORT": "9529",
        "DD_SERVICE": "{fieldRef:metadata.labels['app']}",
        "POD_NAME": "{fieldRef:metadata.name}",
        "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
        "NODE_NAME": "{fieldRef:spec.nodeName}",
        "DD_TAGS": "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
    }
}
```

> 버전 선택 기준은 DDTrace 업스트림 패키지의 런타임 요구 사항입니다. Node.js는 npm `dd-trace`의 `engines.node`를 기준으로 하며, Python은 PyPI `ddtrace`의 `Requires-Python` 및 wheel 지원을 기준으로 합니다.

#### Java 주입 검증 {#ddtrace-java-injection}

Java 주입은 라이브러리 볼륨 외에도 JVM이 `-javaagent`를 실제로 로드하도록 해야 합니다. 애플리케이션을 시작한 후 최종 Pod의 `JAVA_TOOL_OPTIONS`, 컨테이너 command/args 또는 프로세스 명령줄을 검사하여 Operator가 주입한 Agent 경로가 포함되어 있는지 확인합니다. 그런 다음 `DD_AGENT_HOST`와 `DD_TRACE_AGENT_PORT=9529`를 확인합니다. `datakit-lib-init`의 성공만으로는 JVM이 Agent를 로드했다고 판단할 수 없습니다.

### `check_annotation` 설정 항목 설명 {#check-annotation-config}

`check_annotation`는 DataKit Operator가 Pod의 **버전 어노테이션**(예: `admission.datakit/java-lib.version`, `admission.datakit/python-lib.version`, `admission.datakit/nodejs-lib.version`)을 처리하는 방식을 제어하는 중요한 설정 필드입니다. 이 필드의 값과 동작은 다음과 같습니다.

| 값     | 동작 설명                                                                 |
|----------|--------------------------------------------------------------------------|
| `false`  | **(기본값)** Pod의 **버전 어노테이션** 검사를 무시하고 선택기 규칙에 따라 직접 주입 |
| `true`   | **버전 어노테이션** 검사를 활성화하며, 일치하는 Pod에 버전 어노테이션이 있는 경우에만 주입 |

#### 주요 로직 설명 {#important-logic}

1. **기능별 어노테이션은 항상 적용됩니다**:
   - `admission.datakit/ddtrace.enabled`는 `check_annotation` 설정의 **영향을 받지 않습니다**
   - `check_annotation`가 `true`인지 `false`인지와 관계없이 `admission.datakit/ddtrace.enabled`를 검사합니다
   - `admission.datakit/ddtrace.enabled: "false"`이면 주입을 즉시 거부합니다

2. **버전 어노테이션은 `check_annotation`의 제어를 받습니다**:
   - `admission.datakit/<language>-lib.version`는 `check_annotation` 설정의 **영향을 받습니다**
   - `check_annotation: true`이면 버전 어노테이션이 있어야 주입합니다
   - `check_annotation: false`이면 버전 어노테이션 검사를 무시합니다

3. **전역 어노테이션은 항상 적용됩니다**:
   - `admission.datakit/enabled`는 `check_annotation` 설정의 **영향을 받지 않습니다**
   - `admission.datakit/enabled: "false"`이면 모든 주입을 완전히 거부합니다(최우선 순위)

지원되는 DDTrace 관련 Annotation:

| Annotation                           | 기능 설명                               | 값             | `check_annotation`의 영향 여부 | 설명                                                                 |
|--------------------------------------|----------------------------------------|------------------|---------------------------|----------------------------------------------------------------------|
| `admission.datakit/ddtrace.enabled`  | DDTrace 주입 제어                     | `"true"`/`"false"` | **아니요**                   | `"true"`: 주입 허용, `"false"`: 주입 거부, 미설정: 규칙 일치 여부에 따라 결정    |
| `admission.datakit/java-lib.version` | DDTrace Java Agent 버전 지정          | 버전 문자열       | **예**                    | 예: `"1.12.0"`, 설정의 기본 이미지 버전을 재정의하는 데 사용                        |
| `admission.datakit/python-lib.version` | DDTrace Python Lib 버전 지정        | 버전 문자열       | **예**                    | 예: `"v3.19.7"`, 설정의 기본 이미지 버전을 재정의하는 데 사용                       |
| `admission.datakit/nodejs-lib.version` | DDTrace Node.js Lib 버전 지정       | 버전 문자열       | **예**                    | 예: `"5.102.0"`, 설정의 기본 이미지 버전을 재정의하는 데 사용                       |
| `admission.datakit/enabled`          | 모든 주입 기능 제어(최우선 순위)         | `"true"`/`"false"` | **아니요**                   | `"false"`: 모든 주입을 완전히 거부하며 우선순위가 가장 높음                            |

#### `check_annotation: true`인 경우 {#when-check-annotation-true}

다음 조건을 모두 충족해야 주입을 수행합니다.

1. **설정 일치**: Pod가 `namespace_selectors` 및 `label_selectors` 규칙과 일치해야 합니다
2. **기능 어노테이션 허용**: `admission.datakit/ddtrace.enabled`가 `"false"`가 아니어야 합니다(존재하는 경우)
3. **버전 어노테이션 존재**: Pod에 버전 어노테이션(예: `admission.datakit/java-lib.version`, `admission.datakit/python-lib.version` 또는 `admission.datakit/nodejs-lib.version`)이 있어야 합니다

#### `check_annotation: false`인 경우 {#when-check-annotation-false}

다음 조건을 충족해야 주입을 수행합니다.

1. **설정 일치**: Pod가 `namespace_selectors` 및 `label_selectors` 규칙과 일치해야 합니다
2. **기능 어노테이션 허용**: `admission.datakit/ddtrace.enabled`가 `"false"`가 아니어야 합니다(존재하는 경우)
3. **버전 어노테이션 무시**: 버전 어노테이션이 없어도 주입합니다

#### 사용 사례 예시 {#use-case-examples}

1. **엄격한 버전 제어 사용 사례**(`check_annotation: true`):

   ```json
   {
       "namespace_selectors": ["prod"],
       "label_selectors": ["app=backend"],
       "check_annotation": true,
       "image": "internal-registry/dd-lib-java:<pinned-version>",
       "language": "java"
   }
   ```

   **주입 조건**:
   - Pod가 `prod` 네임스페이스에 있고 `app=backend` 레이블이 있어야 합니다
   - Pod에 `admission.datakit/ddtrace.enabled: "false"`가 **없어야 합니다**(존재하는 경우)
   - Pod에 `admission.datakit/java-lib.version` 어노테이션이 **반드시 있어야 합니다**

2. **일괄 주입 사용 사례**(`check_annotation: false`):

   ```json
   {
       "namespace_selectors": ["staging"],
       "label_selectors": ["env=test"],
       "check_annotation": false,
       "image": "internal-registry/dd-lib-java:<pinned-version>",
       "language": "java"
   }
   ```

   **주입 조건**:
   - Pod가 `staging` 네임스페이스에 있고 `env=test` 레이블이 있어야 합니다
   - Pod에 `admission.datakit/ddtrace.enabled: "false"`가 **없어야 합니다**(존재하는 경우)
   - `admission.datakit/java-lib.version` 어노테이션 검사를 **무시합니다**

3. **선택적 거부 사용 사례**:

   ```json
   {
       "namespace_selectors": ["prod"],
       "label_selectors": ["app=java-app"],
       "check_annotation": false,
       "image": "internal-registry/dd-lib-java:<pinned-version>",
       "language": "java"
   }
   ```

   **주입 로직**:
   - 일치하는 모든 Pod에 주입합니다
   - 특정 Pod에 `admission.datakit/ddtrace.enabled: "false"`가 있으면 해당 Pod는 제외됩니다
   - 버전 어노테이션 `admission.datakit/java-lib.version: "1.15.0"`를 사용하여 이미지 버전을 재정의할 수 있지만 주입 여부 결정에는 영향을 주지 않습니다

    다음은 예시입니다.

    ```json
    {
        "namespace_selectors": [],
        "check_annotation": false,
        "label_selectors": [],
        "image": "{{.DDTraceJavaImage}}",
        "language": "java",
        "envs": {
             "DD_AGENT_HOST":           "datakit-service.datakit.svc.cluster.local",
             "DD_TRACE_AGENT_PORT":     "9529",
             "DD_JMXFETCH_STATSD_HOST": "datakit-service.datakit.svc.cluster.local",
             "DD_JMXFETCH_STATSD_PORT": "8125",
             "DD_SERVICE":              "{fieldRef:metadata.labels['service']}",
             "POD_NAME":                "{fieldRef:metadata.name}",
             "POD_NAMESPACE":           "{fieldRef:metadata.namespace}",
             "NODE_NAME":               "{fieldRef:spec.nodeName}",
             "DD_TAGS":                 "pod_name:$(POD_NAME),pod_namespace:$(POD_NAMESPACE),host:$(NODE_NAME)"
         },
        "resources": {
            "requests": {
                "cpu":    "100m",
                "memory": "64Mi"
            },
            "limits": {
                "cpu":    "500m",
                "memory": "512Mi"
            }
        }
    }
    ```

## 특수 Deployment 처리 {#special-deployment}

위의 Operator 설정은 전체 클러스터에 적용되는 DDTrace 주입 설정입니다. 이와 같은 일괄 적용 방식이 특정 Deployment에 적합하지 않은 경우 해당 Deployment에 별도의 Annotation 표시를 추가할 수 있습니다.

Operator는 다음 Annotation을 인식합니다.

- `admission.datakit/ddtrace.enabled`: 개별 Deployment의 주입 활성화 여부를 표시합니다. `"true"`를 입력하면 주입을 활성화하고, `"false"`를 입력하면 주입을 차단합니다. 차단되면 Operator는 해당 Deployment에 대한 주입을 무시합니다
- `admission.datakit/java-lib.version`: 특정 DDTrace Java Agent 버전을 지정합니다
- `admission.datakit/python-lib.version`: 특정 DDTrace Python Lib 버전을 지정합니다
- `admission.datakit/nodejs-lib.version`: 특정 DDTrace Node.js Lib 버전을 지정합니다

> **어노테이션 사용 안내**: `check_annotation` 설정이 버전 어노테이션의 동작에 미치는 영향은 [Annotation 설정 주입](datakit-operator.md#annotation-injection)과 [이 페이지의 `check_annotation` 설정 항목 설명](operator-ddtrace.md#check-annotation-config)을 참조하십시오.

### Annotation 예시 {#anno-demo}

Deployment에 주입 여부 표시를 추가합니다.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app-deployment
  labels:
    app: my-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
      annotations:
        admission.datakit/ddtrace.enabled: "true"
    spec:
      containers:
      - name: my-app
        image: my-app:1.2.3
        ports:
        - containerPort: 80
```

Deployment에 `dd-java-lib`의 특정 버전 번호를 주입합니다[^replace-ddtrace-version].

[^replace-ddtrace-version]: 여기서는 Operator ConfigMap에 있는 동일한 이미지 주소의 버전만 교체합니다. 다른 이미지 주소로 전환할 수 없습니다.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app-deployment
  labels:
    app: my-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
      annotations:
        admission.datakit/java-lib.version: "<version>"
    spec:
      containers:
      - name: my-app
        image: my-app:1.2.3
        ports:
        - containerPort: 80
```

yaml 파일을 사용하여 리소스를 생성합니다.

```shell
$ kubectl apply -f my-app.yaml
...
```

다음과 같이 확인합니다.

```shell
$ kubectl get pod

NAME                                   READY   STATUS    RESTARTS      AGE
my-app-deployment-7bd8dd85f-fzmt2       1/1     Running   0             4s

$ kubectl get pod my-app-deployment-7bd8dd85f-fzmt2 -o=jsonpath={.spec.initContainers\[\*\].name}

datakit-lib-init
```
