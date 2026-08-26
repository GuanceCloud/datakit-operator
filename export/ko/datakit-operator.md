# DataKit Operator

---

:material-kubernetes:

---

DataKit Operator는 Kubernetes Admission Webhook을 통해 새로 생성되는 Pod에 분산 추적, 로그 수집 및 성능 분석 컴포넌트를 주입하고 클러스터 내 Pod 조회 API를 제공합니다.

## 개요 {#overview}

DataKit Operator는 다음 기능을 제공합니다.

| 기능 | 설명 |
| --- | --- |
| DDTrace 자동 주입 | Java, Python, PHP 및 Node.js 지원 |
| OpenTelemetry 자동 주입 | v1.9.0부터 Java, Python 및 Node.js 지원 |
| logfwd 주입 | 사이드카를 통해 컨테이너 표준 출력에 기록되지 않는 파일 로그 수집 |
| Flameshot 주입 | 애플리케이션 Profiling 데이터 동적 수집 |
| Profiler 주입 | 레거시 async-profiler, py-spy 등의 주입 방식과 호환 |
| Logging 구성 주입 | `datakit/logs` Annotation과 해당 파일 볼륨 마운트 추가 |
| Cluster API | 클러스터 내 Pod 데이터 조회를 프록시하여 DataKit의 API Server 직접 접근 부하 감소 |

주입은 Pod `CREATE` 단계에서만 수행되며 이미 실행 중인 Pod는 변경하지 않습니다. Operator 구성, 주입 이미지 또는 워크로드 Annotation을 변경한 후에는 Pod를 다시 생성해야 적용됩니다.

Admission은 fail-open 방식으로 동작합니다. 주입에 실패하면 Operator가 로그를 기록하지만 애플리케이션 Pod 생성을 차단하지 않습니다. 배포 매니페스트의 webhook도 `failurePolicy: Ignore`를 사용합니다.

## 사전 요구 사항 {#prerequisites}

- Kubernetes에서 `admissionregistration.k8s.io/v1`을 지원해야 하며 Kubernetes v1.24 이상을 권장합니다.
- 클러스터에서 `MutatingAdmissionWebhook` Admission Controller를 활성화해야 합니다.
- 클러스터 노드에서 구성된 이미지를 가져올 수 있어야 합니다. 오프라인 환경에서는 이미지를 미리 프라이빗 레지스트리에 동기화하십시오.

## 설치 {#install}

<!-- markdownlint-disable MD046 -->
=== "Deployment"

    [*datakit-operator.yaml*](https://static.<<<custom_key.brand_main_domain>>>/datakit-operator/datakit-operator.yaml){:target="_blank"}을 다운로드하여 설치합니다.

    ```shell
    kubectl create namespace datakit
    wget https://static.<<<custom_key.brand_main_domain>>>/datakit-operator/datakit-operator.yaml
    kubectl apply -f datakit-operator.yaml
    kubectl get pod -n datakit
    ```

    Pod가 정상적으로 시작되면 `Running`으로 표시됩니다.

    ```text
    NAME                                READY   STATUS    RESTARTS   AGE
    datakit-operator-f948897fb-5w5nm    1/1     Running   0          15s
    ```

=== "Helm"

    Helm 3.0 이상이 필요합니다.

    ```shell
    helm install datakit-operator datakit-operator \
        <<<% if custom_key.brand_key == 'guance' -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/datakit-operator \
        <<<% else -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/truewatch \
        <<<% endif -%>>>
        -n datakit --create-namespace
    ```

    배포 상태를 확인합니다.

    ```shell
    helm -n datakit list
    ```

    업그레이드합니다.

    ```shell
    helm -n datakit get values datakit-operator -a -o yaml > values.yaml
    helm upgrade datakit-operator datakit-operator \
        <<<% if custom_key.brand_key == 'guance' -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/datakit-operator \
        <<<% else -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/truewatch \
        <<<% endif -%>>>
        -n datakit \
        -f values.yaml
    ```

    제거합니다.

    ```shell
    helm uninstall datakit-operator -n datakit
    ```

???+ attention

    - Operator 프로그램과 배포 매니페스트는 호환되는 버전을 함께 사용해야 합니다. Operator를 업그레이드할 때 YAML 또는 Helm Chart도 함께 업데이트하십시오.
    - `InvalidImageName`이 발생하거나 이미지를 가져오지 못하면 이미지 주소, 레지스트리 권한 및 노드 네트워크를 확인하십시오.
<!-- markdownlint-enable MD046 -->

### 구성 설명 {#jsonconfig}

Operator 구성은 JSON 형식을 사용합니다. 배포 매니페스트는 일반적으로 구성을 ConfigMap에 저장하고 `ENV_JSON_CONFIG` 환경 변수를 통해 로드합니다.

v1.8.0부터 `admission_inject_v2`를 사용합니다.

```json
{
    "server_listen": "0.0.0.0:9543",
    "log_level": "info",
    "admission_inject_v2": {
        "ddtraces": [],
        "otels": [],
        "logfwds": [],
        "flameshots": [],
        "profilers": []
    },
    "admission_mutate": {
        "loggings": []
    }
}
```

위 예시는 구성 구조만 보여 줍니다. 실제 배포 템플릿은 `default` Namespace의 Pod와 일치하는 Java DDTrace 규칙 하나만 유지하며, `otels`가 비어 있으므로 OpenTelemetry 주입은 자동으로 활성화되지 않습니다. 이 기본값은 기존 배포와의 호환성을 유지하기 위한 실행 정책이며 Operator가 Java만 지원한다는 의미는 아닙니다.

Operator는 애플리케이션 컨테이너의 언어를 자동으로 감지하지 않습니다. DDTrace는 Python, PHP 및 Node.js도 지원하고 OpenTelemetry는 Java, Python 및 Node.js를 지원합니다. 이러한 기능을 활성화하려면 [DDTrace 자동 주입](operator-ddtrace.md) 및 [OpenTelemetry 자동 주입](operator-otel.md) 문서에 따라 상호 배타적인 언어 Label을 사용하는 규칙을 추가하십시오.

레거시 `admission_inject` 구성도 계속 지원됩니다. 이전 구성에서 유효한 `ddtrace`, `logfwd` 또는 `profiler`는 각각 해당 v2 규칙을 덮어씁니다. 업그레이드할 때 두 구성에 유효한 규칙을 동시에 관리하지 마십시오.

## Cluster API {#cluster-api}

DataKit Operator [:octicons-tag-24: v1.8.1](operator-changelog.md#cl-1.8.1) 이상에서는 Cluster API를 제공합니다. 이 API는 Operator의 Pod informer 캐시를 사용하여 Pod 데이터 조회를 프록시하므로 각 DataKit 인스턴스가 API Server에 직접 접근할 때 발생하는 부하를 줄입니다.

Cluster API는 기본적으로 활성화됩니다. Operator는 시작할 때 ServiceAccount의 Pod 읽기 권한을 확인합니다. 권한이 부족하거나 캐시 동기화에 실패하면 관련 경로를 등록하지 않지만 다른 기능은 계속 실행됩니다. 필요한 최소 권한은 다음과 같습니다.

```yaml
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
```

API는 Operator Service를 재사용하며 기본 주소는 `https://datakit-operator.datakit.svc:443`입니다.

| API | 설명 |
| --- | --- |
| `/v1/cluster/api/v1/pods` | 모든 Pod 조회 |
| `/v1/cluster/api/v1/namespaces/{namespace}/pods` | 지정한 Namespace의 Pod 조회 |
| `/v1/cluster/api/v1/namespaces/{namespace}/pods/{name}` | 지정한 Pod 조회 |

[:octicons-tag-24: v1.8.9](operator-changelog.md#cl-1.8.9)부터 `view=ebpf-v1`을 사용하여 eBPF용으로 간소화된 Pod 구조를 가져올 수 있습니다.

```shell
curl -k "https://datakit-operator.datakit.svc:443/v1/cluster/api/v1/pods?view=ebpf-v1"
```

## 주입 규칙 {#datakit-operator-inject}

각 주입 구성은 먼저 selector로 Pod와 일치해야 합니다. Annotation은 주입을 거부하거나 `check_annotation`이 활성화된 경우 주입 대상을 추가로 제한할 수 있지만, selector 없이 단독으로 주입을 트리거할 수는 없습니다.

### Selector 구성 {#selectors-injection}

`namespace_selectors`와 `label_selectors`는 모두 배열입니다.

- 같은 배열의 여러 selector는 OR로 평가합니다.
- 두 배열을 모두 구성하면 namespace와 label 조건이 모두 일치해야 합니다.
- 주입 규칙에 한 조건만 구성하면 해당 조건만 사용하여 일치 여부를 판단합니다. 두 조건을 모두 구성하지 않으면 현재 규칙을 비활성화하고 이후 규칙 검사를 계속합니다.
- Logging mutation 규칙에는 두 조건이 모두 필요합니다. 한 조건이라도 없으면 현재 규칙을 비활성화합니다.

`namespace_selectors`는 Go 정규식을 사용합니다. 문자열 `"*"`는 모든 Namespace와 일치하는 축약형입니다. 정확히 일치시키려면 `^`와 `$`를 사용하는 것이 좋습니다. `label_selectors`는 Kubernetes Label Selector 구문을 사용하며 `=`, `==` 및 `!=`에 glob 일치 기능이 확장되어 있습니다.

빈 문자열, 공백으로만 된 값, 잘못된 Namespace 정규식, 잘못되었거나 제약 조건이 비어 있는 Label selector는 유효하지 않은 항목입니다. Operator는 시작할 때 warning을 기록하고 각 잘못된 항목을 무시합니다. 구성된 조건에 유효한 항목이 하나도 남지 않으면 현재 규칙을 비활성화하고 이후 규칙 검사를 계속합니다. 모든 Namespace와 명시적으로 일치시키려면 `"*"`를 사용하십시오.

다음 규칙은 `production` Namespace에서 `admission.datakit/ddtrace-language=java` Label이 있는 Pod에만 일치합니다.

```json
{
    "namespace_selectors": ["^production$"],
    "label_selectors": ["admission.datakit/ddtrace-language=java"]
}
```

하나의 Pod가 여러 DDTrace 또는 OTel 규칙에 일치할 수 있습니다. Operator는 구성 순서에 따라 Annotation 조건을 충족하는 첫 번째 규칙을 선택합니다. 선택된 규칙의 언어, 이미지 또는 기타 구성이 잘못되어도 이후 규칙으로 대체하지 않습니다. 여러 언어의 규칙에는 상호 배타적인 Label을 사용하십시오.

DDTrace와 OTel이 동시에 일치하면 DDTrace가 우선합니다. DDTrace 주입에 실패해도 OTel로 대체하지 않습니다.

### Annotation 구성 {#annotation-injection}

Annotation은 Pod 또는 Deployment 같은 컨트롤러의 `.spec.template.metadata.annotations`에 추가해야 합니다.

| Annotation | 역할 |
| --- | --- |
| `admission.datakit/enabled` | Operator의 모든 Pod 변경 제어. 우선순위가 가장 높습니다. |
| `admission.datakit/ddtrace.enabled` | DDTrace 주입 제어 |
| `admission.datakit/otel.enabled` | OTel 주입 제어 |
| `admission.datakit/logfwd.enabled` | logfwd 주입 제어 |
| `admission.datakit/flameshot.enabled` | Flameshot 주입 제어 |
| `admission.datakit/profiler.enabled` | 레거시 Profiler 주입 제어 |

이 활성화 Annotation은 관대한 방식으로 처리됩니다. `false`로 파싱되는 값만 해당 기능을 비활성화하며, Annotation이 없거나 값을 파싱할 수 없으면 `true`로 처리합니다. 값을 명시적으로 `true`로 설정해도 Pod는 해당 구성 규칙과 일치해야 합니다.

```yaml
metadata:
  annotations:
    admission.datakit/ddtrace.enabled: "false"
    admission.datakit/otel.enabled: "true"
```

### `check_annotation` {#check-annotation-config}

DDTrace, OTel, logfwd 및 레거시 Profiler 규칙은 `check_annotation`을 지원합니다.

| 값 | 동작 |
| --- | --- |
| `false` | 기본값. selector가 일치하면 버전 또는 레거시 구성 Annotation을 요구하지 않습니다. |
| `true` | selector가 일치하고 해당 규칙에 대응하는 Annotation도 있어야 합니다. |

대응 관계는 다음과 같습니다.

| 기능 | `check_annotation: true`일 때 필요한 Annotation |
| --- | --- |
| DDTrace | `admission.datakit/<language>-lib.version` |
| OTel | `admission.datakit/otel-<language>-lib.version` |
| Profiler | `admission.datakit/<language>-profiler.version` |
| logfwd | `admission.datakit/logfwd.instances` |

`check_annotation: true`이고 Pod가 해당 버전 Annotation을 제공하면 DDTrace, OTel 및 Profiler는 규칙의 `image` 태그를 대체하지만 이미지 레지스트리와 이름은 변경하지 않습니다. 기능 활성화 Annotation은 `check_annotation`과 관계없이 항상 적용됩니다.

## 지원되는 주입 기능 {#supported-operator}

| 기능 | 문서 |
| --- | --- |
| DDTrace | [DDTrace 자동 주입](operator-ddtrace.md) |
| OpenTelemetry | [OpenTelemetry 자동 주입](operator-otel.md) |
| logfwd | [logfwd 사이드카 주입](operator-logfwd.md) |
| Flameshot | [Flameshot 주입](operator-flameshot.md) |
| async-profiler | [레거시 Java Profiler 주입](operator-asyncprofile.md) |
| py-spy | [레거시 Python Profiler 주입](operator-pyspy.md) |
| Logging | [로그 수집 구성 주입](operator-logging.md) |

## 환경 변수 값 참조 {#downwardapi}

주입 규칙의 `envs`는 리터럴 값뿐 아니라 자리표시자를 Kubernetes 네이티브 `fieldRef`, `resourceFieldRef` 및 `secretKeyRef`로 변환하는 기능도 지원합니다.

| 형식 | 설명 |
| --- | --- |
| `{fieldRef:metadata.name}` | Pod 이름 |
| `{fieldRef:metadata.namespace}` | Pod Namespace |
| `{fieldRef:metadata.uid}` | Pod UID |
| `{fieldRef:metadata.annotations['<KEY>']}` | 지정한 Pod Annotation |
| `{fieldRef:metadata.labels['<KEY>']}` | 지정한 Pod Label |
| `{fieldRef:spec.serviceAccountName}` | ServiceAccount 이름 |
| `{fieldRef:spec.nodeName}` | 노드 이름 |
| `{fieldRef:status.hostIP}` | 노드 기본 IP |
| `{fieldRef:status.hostIPs}` | 노드 듀얼 스택 IP |
| `{fieldRef:status.podIP}` | Pod 기본 IP |
| `{resourceFieldRef:limits.cpu}` | 첫 번째 애플리케이션 컨테이너의 CPU limit. 단위는 millicore입니다. |
| `{resourceFieldRef:limits.memory}` | 첫 번째 애플리케이션 컨테이너의 메모리 limit. 단위는 MiB입니다. |
| `{resourceFieldRef:requests.cpu}` | 첫 번째 애플리케이션 컨테이너의 CPU request. 단위는 millicore입니다. |
| `{resourceFieldRef:requests.memory}` | 첫 번째 애플리케이션 컨테이너의 메모리 request. 단위는 MiB입니다. |
| `{secretKeyRef:<SECRET_NAME>.<KEY>}` | Pod가 속한 Namespace의 Secret key |

환경 변수는 구성 순서를 유지하므로 뒤에 오는 값에서 Kubernetes `$(VAR)` 구문으로 앞서 정의한 변수를 참조할 수 있습니다.

```json
{
    "envs": {
        "POD_NAME": "{fieldRef:metadata.name}",
        "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
        "RESOURCE_TAGS": "pod_name=$(POD_NAME),pod_namespace=$(POD_NAMESPACE)"
    }
}
```

인식할 수 없는 자리표시자는 일반 문자열로 주입됩니다. `resourceFieldRef`는 첫 번째 애플리케이션 컨테이너만 참조합니다. 해당 컨테이너에 관련 request 또는 limit가 선언되어 있지 않으면 그 환경 변수는 주입되지 않습니다.

### `{secretKeyRef:*}` {#secretkeyref}

Secret 참조 형식은 다음과 같습니다.

```text
{secretKeyRef:<secret-name>.<key>}
```

예:

```json
{
    "envs": {
        "ACCESS_KEY_ID": "{secretKeyRef:flameshot-oss.access_key_id}",
        "ACCESS_KEY_SECRET": "{secretKeyRef:flameshot-oss.access_key_secret}"
    }
}
```

Operator는 Secret을 읽지 않으며 Kubernetes가 컨테이너 시작 시 참조를 해석합니다. Secret은 애플리케이션 Pod와 같은 Namespace에 있어야 합니다. Secret 또는 key가 없으면 Pod가 `CreateContainerConfigError` 상태가 됩니다. `.`이 Secret 이름과 key의 구분자로 사용되므로 이 형식에서 Secret 이름에 `.`을 포함할 수 없습니다.

## FAQ {#faq}

### 특정 Pod의 모든 변경을 비활성화하려면 어떻게 해야 하나요? {#disable-inject}

Pod 템플릿에 다음을 추가합니다.

```yaml
admission.datakit/enabled: "false"
```

### 주입이 적용되지 않는 이유는 무엇인가요? {#debug}

다음 순서로 확인합니다.

1. 구성 업데이트 후 Pod를 새로 생성했는지 확인합니다.
1. Namespace와 Label이 같은 규칙에 일치하는지 확인합니다.
1. `admission.datakit/enabled` 및 기능 활성화 Annotation이 `false`인지 확인합니다.
1. `check_annotation: true`이면 올바른 버전 또는 구성 Annotation이 있는지 확인합니다.
1. Operator 로그에서 selector, 이미지, 환경 변수 또는 SecurityContext 충돌 warning을 확인합니다.

### AWS EKS 환경에서 주의할 점은 무엇인가요? {#aws-eks}

EKS 컨트롤 플레인에서 Operator webhook의 `9543` 포트에 접근할 수 있어야 합니다. 주입이 적용되지 않으면 클러스터 보안 그룹과 네트워크 정책이 해당 방향의 트래픽을 허용하는지 확인하십시오.
