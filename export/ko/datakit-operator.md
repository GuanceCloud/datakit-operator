# DataKit Operator

---

:material-kubernetes:

---

DataKit Operator는 Kubernetes 오케스트레이션 환경에서 DataKit과 연동되는 프로젝트로, DataKit을 더욱 편리하게 배포하고 검증 및 주입 등의 기능을 사용할 수 있도록 지원합니다.

## 개요 {#overview}

DataKit Operator는 Kubernetes Admission Controller 메커니즘을 통해 Kubernetes 클러스터에 자동 주입 기능을 제공하여 사용자가 관측 기능을 더욱 쉽게 통합할 수 있도록 지원합니다. 주요 기능은 다음과 같습니다.

- **DDTrace 주입**: Java 애플리케이션에 APM 추적 에이전트를 자동으로 주입
- **로그 수집**: logfwd Sidecar를 통해 컨테이너 로그를 자동으로 수집
- **성능 분석**: Flameshot 또는 Profiler 컴포넌트를 주입하여 애플리케이션 성능을 모니터링
- **구성 관리**: 전역 구성과 선언적 구성의 두 가지 주입 방식 지원
- **Cluster API**: DataKit 등의 컴포넌트가 Kubernetes 메타데이터를 가져올 수 있도록 클러스터 내 Pod 쿼리 프록시 제공

**핵심 이점**:

- **자동 배포**: 애플리케이션 YAML을 수동으로 수정할 필요가 없어 구성 오류 감소
- **일괄 관리**: Namespace 및 Label Selector를 통한 일괄 주입
- **유연한 구성**: JSON 구성 및 Annotation을 통한 세밀한 제어 지원
- **버전 호환성**: 하위 호환성을 유지하여 원활한 업그레이드 지원

## 사전 요구 사항 {#prerequisites}

- Kubernetes v1.24.1 이상을 권장하며, 인터넷에 연결할 수 있어야 합니다(yaml 파일 다운로드 및 해당 이미지 가져오기).
- `MutatingAdmissionWebhook` 및 `ValidatingAdmissionWebhook` [컨트롤러](https://kubernetes.io/zh-cn/docs/reference/access-authn-authz/extensible-admission-controllers/#prerequisites){:target="_blank"}가 활성화되어 있는지 확인합니다.
- `admissionregistration.k8s.io/v1` API가 활성화되어 있는지 확인합니다.

## 설치 {#install}

<!-- markdownlint-disable MD046 -->
=== "Deployment"

    [*datakit-operator.yaml*](https://static.<<<custom_key.brand_main_domain>>>/datakit-operator/datakit-operator.yaml){:target="_blank"}을 다운로드합니다. 단계는 다음과 같습니다.


    ``` shell
    $ kubectl create namespace datakit
    $ wget https://static.<<<custom_key.brand_main_domain>>>/datakit-operator/datakit-operator.yaml
    $ kubectl apply -f datakit-operator.yaml
    $ kubectl get pod -n datakit


    NAME                               READY   STATUS    RESTARTS   AGE
    datakit-operator-f948897fb-5w5nm   1/1     Running   0          15s
    ```

=== "Helm"

    사전 요구 사항

    * Kubernetes >= 1.14
    * Helm >= 3.0+

    ```shell
    $ helm install datakit-operator datakit-operator \
        <<<% if custom_key.brand_key == 'guance' -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/datakit-operator \
        <<<% else -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/truewatch \
        <<<% endif -%>>>
        -n datakit --create-namespace
    ```

    배포 상태를 확인합니다.

    ```shell
    $ helm -n datakit list
    ```

    다음 명령으로 업그레이드할 수 있습니다.

    ```shell
    $ helm -n datakit get values datakit-operator -a -o yaml > values.yaml
    $ helm upgrade datakit-operator datakit-operator \
        <<<% if custom_key.brand_key == 'guance' -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/datakit-operator \
        <<<% else -%>>>
        --repo https://pubrepo.<<<custom_key.brand_main_domain>>>/chartrepo/truewatch \
        <<<% endif -%>>>
        -n datakit \
        -f values.yaml
    ```

    다음 명령으로 제거할 수 있습니다.

    ```shell
    $ helm uninstall datakit-operator -n datakit
    ```

???+ attention

    - DataKit Operator는 프로그램과 yaml 간의 버전 대응 관계가 엄격합니다. 너무 오래된 yaml을 사용하면 최신 버전의 DataKit Operator를 설치하지 못할 수 있으므로 최신 yaml을 다시 다운로드하십시오.
    - `InvalidImageName` 오류가 발생하면 이미지를 수동으로 pull할 수 있습니다.
<!-- markdownlint-enable MD046 -->

### 구성 설명 {#jsonconfig}

DataKit Operator 구성은 JSON 형식이며, Kubernetes에서는 별도의 ConfigMap에 저장되고 환경 변수를 통해 컨테이너에 로드됩니다.

<!-- markdownlint-disable MD046 -->
=== "DataKit Operator >= v1.8.0"

    DataKit-Operator v1.8.0부터 `admission_inject_v2` 구성 항목 사용을 권장합니다. 새로운 구성은 배열 구조를 사용하여 더욱 유연한 구성 방식을 지원합니다.

    ```json
    {
        "server_listen": "0.0.0.0:9543", // Operator 자체 서비스 수신 주소
        "log_level": "info",             // Operator 자체 로그 레벨
        "admission_inject_v2": {         // 주입 구성 v2
            "ddtraces": [...],           // DDTrace 구성 배열
            "logfwds": [...],            // 로그 전달 구성 배열
            "flameshots": [...]          // 성능 분석 구성 배열
        },
        "admission_mutate": {            // 구성 변경
            "loggings": [...]            // 로그 구성 변경
        }
    }
    ```

=== "DataKit Operator < v1.8.0"

    ```json
    {
        "server_listen": "0.0.0.0:9543",
        "log_level":     "info",
        "admission_inject": {
            "ddtrace": {...},
            "profiler": {...},
            "logfwd": {...}
        },
        "admission_mutate": {
            "loggings": [...]
        }
    }
    ```
<!-- markdownlint-enable MD046 -->

## Cluster API {#cluster-api}

DataKit Operator [:octicons-tag-24: v1.8.1](operator-changelog.md#cl-1.8.1) 이상 버전에서는 클러스터 내부에서 Pod 데이터를 프록시 조회하는 Cluster API를 제공합니다. DataKit은 이 인터페이스를 통해 Kubernetes 메타데이터를 가져와 각 DataKit 인스턴스가 API Server에 직접 액세스하면서 발생하는 부하를 줄일 수 있습니다.

Cluster API는 기본적으로 활성화되며 별도의 구성 스위치가 필요하지 않습니다. Operator는 시작할 때 자체 ServiceAccount에 Pod 읽기 권한이 있는지 확인합니다. 권한이 없으면 로그에 RBAC 검사 실패 메시지를 기록하고 Cluster API 관련 라우트를 비활성화합니다. 최신 `datakit-operator.yaml` 또는 Helm Chart에는 필요한 권한이 포함되어 있습니다. 이전 버전의 YAML에서 업그레이드하는 경우 ClusterRole에 최소한 다음 권한이 포함되어 있는지 확인해야 합니다.

```yaml
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
```

인터페이스 주소는 Operator Service를 재사용하며 기본 주소는 `https://datakit-operator.datakit.svc:443`입니다. 현재 지원되는 Pod 쿼리 인터페이스는 다음과 같습니다.

| 인터페이스 | 설명 |
| --- | --- |
| `/v1/cluster/api/v1/pods` | 전체 Pod 목록 조회 |
| `/v1/cluster/api/v1/namespaces/{namespace}/pods` | 지정된 Namespace의 Pod 목록 조회 |
| `/v1/cluster/api/v1/namespaces/{namespace}/pods/{name}` | 지정된 Pod 조회 |

DataKit Operator [:octicons-tag-24: v1.8.9](operator-changelog.md#cl-1.8.9)부터 Pod 쿼리 인터페이스에서 `view=ebpf-v1` 쿼리 파라미터를 지원합니다. 이 뷰는 eBPF에 필요하지 않은 대용량 필드를 제거하고 워크로드 식별에 필요한 Pod 기본 정보만 유지하여 대규모 클러스터 환경에서 JSON 전송 및 파싱 오버헤드를 줄입니다.

```shell
curl -k "https://datakit-operator.datakit.svc:443/v1/cluster/api/v1/pods?view=ebpf-v1"
```

## 주입 방식 {#datakit-operator-inject}

DataKit Operator는 다음 두 가지 리소스 입력 방식을 지원합니다.

1. selector 구성 주입(명령형)

    DataKit-Operator 구성을 수정하여 대상 Pod의 Namespace와 Selector를 지정합니다. 조건에 맞는 Pod가 발견되면 주입을 실행합니다.

    **장점**: 대상 Pod에 Annotation을 추가할 필요가 없습니다. 단, 대상 Pod를 재시작해야 합니다.

    **단점**: 범위가 충분히 정밀하지 않아 불필요한 주입이 발생할 수 있습니다.

1. Annotation 구성 주입(선언형)

    대상 Pod에 Annotation을 추가하여 자체 주입을 활성화합니다.

    **장점**: Annotation을 통해 주입 거부 여부를 정밀하게 제어할 수 있습니다.

    **단점**: Annotation만으로 주입을 트리거할 수 없으며 일치 규칙도 구성해야 합니다. 즉, 대상 Pod의 annotation에서 주입을 활성화하는 것 외에도 Operator의 다른 필드를 구성해야 합니다.

### Selector 구성 주입 {#selectors-injection}

`namespace_selectors` 및 `label_selectors`을 구성하여 일괄 주입할 수 있습니다.

`admission_inject_v2` 구성에서는 `namespace_selectors` 및 `label_selectors`을 배열 항목에 직접 구성합니다. DDTrace 주입을 예로 들면 다음과 같습니다.

```json
{
    "admission_inject_v2": {
        "ddtraces": [
            {
                "namespace_selectors": ["testns"],
                "label_selectors":     ["app=log-output"],
                ...
            }
        ]
    }
}
```

- `namespace_selectors`: Namespace Selector 배열로, 정규식 일치를 지원합니다. 정확히 일치시키려면 `^` 및 `$`으로 패턴을 감싸십시오. 예: `^testns$`
- `label_selectors`: Label Selector 배열로, Kubernetes Label Selector 구문을 사용합니다.

두 selector를 모두 구성한 경우 대상 Pod는 두 조건을 모두 충족해야 합니다. label selector 작성 규칙은 이 [공식 문서](https://kubernetes.io/zh-cn/docs/concepts/overview/working-with-objects/labels/#label-selectors){:target="_blank"}를 참고하십시오.

### Annotation 구성 주입 {#annotation-injection}

Deployment에 지정된 Annotation을 추가하여 주입 허용 여부를 제어할 수 있습니다. Annotation은 template에 추가해야 합니다.

지원되는 Annotation은 다음과 같습니다.

| Annotation                            | 기능 설명            | 값             | 우선순위   |
| ------------                          | ----------          | ------           | -------- |
| `admission.datakit/ddtrace.enabled`   | ddtrace 주입 제어   | `"true"/"false"` | 중간       |
| `admission.datakit/logfwd.enabled`    | logfwd 주입 제어    | `"true"/"false"` | 중간       |
| `admission.datakit/flameshot.enabled` | flameshot 주입 제어 | `"true"/"false"` | 중간       |
| `admission.datakit/enabled`           | 모든 주입 기능 제어    | `"true"/"false"` | **최고** |

예:

```yaml
    annotations:
    admission.datakit/ddtrace.enabled: "true"
    admission.datakit/logfwd.enabled: "true"
```

<!-- markdownlint-disable MD046 -->
???+ tip

    Annotation을 사용하여 주입을 거부할 수 있습니다(`"false"`으로 설정). 다만 능동적으로 주입하려면 다음과 같이 구성해야 합니다.

    1. DataKit-Operator 구성에서 일치 규칙(`namespace_selectors`/`label_selectors`)과 해당 구성 필드를 설정합니다.
    1. Pod가 구성의 selectors와 일치하도록 합니다.
<!-- markdownlint-enable MD046 -->

### `check_annotation` 구성 항목 설명 {#check-annotation-config}

`check_annotation`은 DataKit Operator가 Pod의 **버전 Annotation**을 처리하는 방식을 제어하는 구성 필드입니다. 이 필드의 값과 동작은 다음과 같습니다.

| 값     | 동작 설명                                                                 |
|----------|--------------------------------------------------------------------------|
| `false`  | **(기본값)** Pod의 **버전 Annotation** 검사를 무시하고 Selector 규칙에 따라 바로 주입 |
| `true`   | **버전 Annotation** 검사를 활성화하고 일치하는 Pod에 버전 Annotation이 있는 경우에만 주입             |

#### Annotation 유형 설명 {#annotation-types}

DataKit Operator는 동작이 서로 다른 두 가지 유형의 Annotation을 지원합니다.

**1. 활성화/비활성화 Annotation(`check_annotation`의 영향을 받지 않음)**
이 Annotation은 특정 기능의 활성화 또는 비활성화 여부를 제어하며, **`check_annotation` 구성의 영향을 받지 않습니다**.

| Annotation                            | 기능 설명            | 값             | 우선순위   |
| ------------                          | ----------          | ------           | -------- |
| `admission.datakit/enabled`           | 모든 주입 기능 제어    | `"true"/"false"` | **최고** |
| `admission.datakit/ddtrace.enabled`   | ddtrace 주입 제어   | `"true"/"false"` | 중간       |
| `admission.datakit/logfwd.enabled`    | logfwd 주입 제어    | `"true"/"false"` | 중간       |
| `admission.datakit/flameshot.enabled` | flameshot 주입 제어 | `"true"/"false"` | 중간       |

**2. 버전 Annotation(`check_annotation`의 영향을 받음)**
이 Annotation은 컴포넌트 버전을 지정하며, **`check_annotation` 구성의 제어를 받습니다**.

| Annotation                                  | 기능 설명                       | 값            |
| --------------------------------------      | ------------------------------ | ------------    |
| `admission.datakit/java-lib.version`        | DDTrace Java Agent 버전 지정   | 버전 문자열      |
| `admission.datakit/python-lib.version`      | DDTrace Python Agent 버전 지정 | 버전 문자열      |
| `admission.datakit/java-profiler.version`   | Java Profiler 버전 지정        | 버전 문자열      |
| `admission.datakit/python-profiler.version` | Python Profiler 버전 지정      | 버전 문자열      |
| `admission.datakit/golang-profiler.version` | Golang Profiler 버전 지정      | 버전 문자열      |
| `admission.datakit/logfwd.instances`        | logfwd Sidecar 버전 지정       | JSON 구성 문자열 |

#### 주입 로직 설명 {#injection-logic}

**핵심 규칙**:

- `admission.datakit/enabled:"false"`은 모든 주입을 거부합니다(최고 우선순위).
- 기능별 활성화 Annotation(예: `admission.datakit/ddtrace.enabled: "false"`)은 해당 기능의 주입을 거부합니다.
- `check_annotation: true`인 경우 해당 버전 Annotation이 있어야 주입됩니다.
- `check_annotation: false`인 경우 버전 Annotation 검사를 무시합니다.

**주입 조건 비교**:

| 조건                          | `check_annotation: true` | `check_annotation: false` |
|-------------------------------|--------------------------|---------------------------|
| 구성 일치(selector 규칙)     | ✓ 반드시 충족              | ✓ 반드시 충족               |
| 활성화 Annotation이 `"false"`이 아님        | ✓ 반드시 충족              | ✓ 반드시 충족               |
| 버전 Annotation 존재                  | ✓ 반드시 존재              | ✗ 무시 가능                 |

#### 사용 사례 예시 {#use-case-examples}

1. **일괄 주입**(`check_annotation: false`):
   다수의 Pod에 자동으로 주입해야 하는 사용 사례에 적합하며, 각 Pod에 버전 Annotation을 추가할 필요가 없습니다.

2. **정밀 제어**(`check_annotation: true`):
   버전을 엄격하게 제어하고 명시적으로 표시된 Pod에만 주입해야 하는 사용 사례에 적합합니다.



## 지원되는 주입 기능 목록 {#supported-operator}

| 기능           | 요약                                                                                      |
| ---            | ---                                                                                       |
| DDtrace Agent  | DDTrace 컴포넌트 주입, [여기](operator-ddtrace.md) 참조                                        |
| logfwd         | 컨테이너 내부 로그를 수집하는 logfwd 컴포넌트 주입, [여기](operator-logfwd.md) 참조                           |
| Flameshot      | 애플리케이션 프로파일링을 동적으로 수집하는 Flameshot 컴포넌트 주입, [여기](operator-flameshot.md) 참조             |
| async-profiler | Java 애플리케이션의 프로파일링을 주기적으로 수집하는 async-profiler 주입, [여기](operator-asyncprofile.md) 참조 |
| py-spy         | Python 애플리케이션의 프로파일링을 수집하는 py-spy 주입, [여기](operator-pyspy.md) 참조                  |
| logging        | 로그 수집 구성 주입, [여기](operator-logging.md) 참조                                        |

## 환경 변수 값 참조 {#downwardapi}

DataKit Operator의 `envs`은 자리표시자를 Kubernetes 네이티브 환경 변수 값 참조로 변환할 수 있습니다. `fieldRef`은 [:octicons-tag-24: v1.4.2](operator-changelog.md#cl-1.4.2)부터 지원되며, 구체적인 필드는 Kubernetes [Downward API](https://kubernetes.io/zh-cn/docs/concepts/workloads/pods/downward-api/#downwardapi-fieldRef)를 참고할 수 있습니다. 현재 지원되는 형식은 다음과 같습니다.

| 필드                                       | 설명                                            | 예시                                 |
| ------:                                    | :------                                         | :------                              |
| `{fieldRef:metadata.name}`                 | Pod 이름                                      | `nginx-123`                          |
| `{fieldRef:metadata.namespace}`            | Pod Namespace                                  | middleware                           |
| `{fieldRef:metadata.uid}`                  | Pod 고유 ID                                   | 12345678-1234-1234-1234-123456789abc |
| `{fieldRef:metadata.annotations['<KEY>']}` | Pod Annotation `<KEY>`의 값                         | metadata.annotations['myannotation'] |
| `{fieldRef:metadata.labels['<KEY>']}`      | Pod Label `<KEY>`의 값                         | metadata.labels['app']               |
| `{fieldRef:spec.serviceAccountName}`       | Pod ServiceAccount 이름                              | default                              |
| `{fieldRef:spec.nodeName}`                 | Pod가 실행되는 노드 이름                        | node-01                              |
| `{fieldRef:status.hostIP}`                 | Pod가 위치한 노드의 기본 IP 주소                        | 192.168.1.1                          |
| `{fieldRef:status.hostIPs}`                | status.hostIP의 듀얼 스택 버전                    | ["192.168.1.1", "2001:db8::1"]       |
| `{fieldRef:status.podIP}`                  | Pod의 기본 IP 주소                                | 10.0.0.1                             |
| `{resourceFieldRef:limits.cpu}`            | Pod 첫 번째 컨테이너의 CPU Limit(단위: millicores)   | 500                                  |
| `{resourceFieldRef:limits.memory}`         | Pod 첫 번째 컨테이너의 Memory Limit(단위: MiB)       | 1024                                 |
| `{resourceFieldRef:requests.cpu}`          | Pod 첫 번째 컨테이너의 CPU Request(단위: millicores) | 200                                  |
| `{resourceFieldRef:requests.memory}`       | Pod 첫 번째 컨테이너의 Memory Request(단위: MiB)     | 512                                  |
| `{secretKeyRef:<SECRET_NAME>.<KEY>}`       | Pod가 위치한 namespace의 Secret key 참조         | `{secretKeyRef:flameshot-oss.access_key_id}` |

예를 들어 Pod 이름이 `nginx-123`이고 namespace가 `middleware`일 때 환경 변수 `POD_NAME` 및 `POD_NAMESPACE`을 주입하려면 다음을 참고하십시오.

```json
{
    "admission_inject_v2": {
        "ddtraces": [
            {
                "namespace_selectors": ["middleware"],
                "language":            "java",
                "image":               "dd-lib-java-init:latest",
                "envs": {
                    "POD_NAME":      "{fieldRef:metadata.name}",
                    "POD_NAMESPACE": "{fieldRef:metadata.namespace}"
                }
            }
        ]
    }
}
```

결과적으로 해당 Pod에서 다음을 확인할 수 있습니다.

``` shell
$ env | grep POD
POD_NAME=nginx-123
POD_NAMESPACE=middleware
```

<!-- markdownlint-disable MD046 -->
???+ note

    Value 자리표시자를 인식할 수 없으면 일반 문자열로 환경 변수에 추가됩니다. 예를 들어 `"POD_NAME": "{fieldRef:metadata.PODNAME}"`은 잘못된 형식이며, 환경 변수에는 `POD_NAME={fieldRef:metadata.PODNAME}`으로 설정됩니다.

### `{resourceFieldRef:*}` 관련 중요 설명 {#resourcefieldref-important-notes}

`{resourceFieldRef:*}` 자리표시자는 Pod의 **첫 번째 컨테이너**에 설정된 리소스 제한(limits)과 요청(requests)을 참조하는 데 사용됩니다. 사용할 때는 다음 사항에 유의하십시오.

1. **리소스 검사**: Pod의 첫 번째 컨테이너에 해당 리소스 제한 또는 요청이 구성되어 있지 않으면 이 자리표시자를 사용하는 환경 변수는 **주입되지 않습니다**. 예:
   - 컨테이너에 `limits.cpu`이 설정되어 있지 않으면 `{resourceFieldRef:limits.cpu}` 환경 변수는 무시됩니다.
   - 컨테이너에 `requests.memory`이 설정되어 있지 않으면 `{resourceFieldRef:requests.memory}` 환경 변수는 무시됩니다.

1. **단위 설명**:
   - CPU 단위는 **millicores (m)**이며, 예를 들어 `500`은 500m(즉, 0.5 CPU)을 나타냅니다.
   - 메모리 단위는 **MiB**이며, 예를 들어 `1024`은 1024Mi(즉, 1GiB)를 나타냅니다.

1. **첫 번째 컨테이너만 지원**: `{resourceFieldRef:*}`은 Pod의 첫 번째 컨테이너 리소스만 참조할 수 있으며 다른 컨테이너의 리소스는 참조할 수 없습니다.

1. **사용 예시**:

```json
{
    "envs": {
        "APP_CPU_LIMIT": "{resourceFieldRef:limits.cpu}",
        "APP_MEMORY_REQUEST": "{resourceFieldRef:requests.memory}"
    }
}
```

1. **검증 방법**: 주입된 Pod의 환경 변수를 확인하여 리소스 자리표시자가 올바르게 파싱되었는지 확인할 수 있습니다.

```shell
kubectl exec <pod-name> -- env | grep APP_
APP_CPU_LIMIT=500
APP_MEMORY_REQUEST=512
```
<!-- markdownlint-enable MD046 -->

### `{secretKeyRef:*}` 관련 설명 {#secretkeyref}

DataKit Operator의 주입 구성에서는 다음 형식을 사용하여 `envs`에서 Kubernetes Secret을 참조할 수 있습니다.

```text
{secretKeyRef:<secret-name>.<key>}
```

예를 들어 Secret `flameshot-oss`에 `access_key_id` 및 `access_key_secret` 두 개의 key가 포함되어 있으면 다음과 같이 구성할 수 있습니다.

```json
{
    "envs": {
        "FLAMESHOT_HPROF_UPLOAD_ACCESS_KEY_ID": "{secretKeyRef:flameshot-oss.access_key_id}",
        "FLAMESHOT_HPROF_UPLOAD_ACCESS_KEY_SECRET": "{secretKeyRef:flameshot-oss.access_key_secret}"
    }
}
```

Operator는 이를 Kubernetes 네이티브 환경 변수 참조로 변환합니다.

```yaml
env:
  - name: FLAMESHOT_HPROF_UPLOAD_ACCESS_KEY_ID
    valueFrom:
      secretKeyRef:
        name: flameshot-oss
        key: access_key_id
  - name: FLAMESHOT_HPROF_UPLOAD_ACCESS_KEY_SECRET
    valueFrom:
      secretKeyRef:
        name: flameshot-oss
        key: access_key_secret
```

사용 시 유의 사항:

1. Secret은 주입 대상 Pod와 동일한 namespace에 있어야 합니다. Operator는 Secret을 읽거나 검사하지 않으며, Secret 파싱은 컨테이너 시작 시 Kubernetes에서 수행합니다.
1. `<secret-name>`은 유효한 Kubernetes Secret 이름이어야 합니다. `.`이 Secret 이름과 key의 구분자로 사용되므로 이 형식의 Secret 이름에는 `.`을 포함할 수 없습니다.
1. `<key>`은 유효한 Kubernetes Secret data key여야 합니다. 길이는 253자를 초과할 수 없고 문자, 숫자, `-`, `_` 또는 `.`만 포함할 수 있으며, `.`이나 `..`일 수 없고 `..`으로 시작할 수 없습니다.
1. Secret 또는 key가 없으면 해당 Secret과 key를 사용할 수 있을 때까지 Pod가 `CreateContainerConfigError` 상태로 전환됩니다.
1. 인식할 수 없거나 검증을 통과하지 못한 표현식은 `secretKeyRef`을 생성하지 않고 일반 문자열로 주입됩니다.

## FAQ {#faq}

### 특정 Pod의 주입을 비활성화하려면 어떻게 해야 합니까? {#disable-inject}

해당 Pod에 Annotation `"admission.datakit/enabled": "false"`을 추가하면 더 이상 어떤 작업도 수행하지 않습니다. 이 설정의 우선순위가 가장 높습니다.

### 작동 원리는 무엇입니까? {#principles}

DataKit-Operator는 Kubernetes Admission Controller 기능을 사용하여 리소스를 주입합니다. 자세한 메커니즘은 [공식 문서](https://kubernetes.io/zh-cn/docs/reference/access-authn-authz/admission-controllers/){:target="_blank"}를 참고하십시오.

### AWS EKS 환경에서 유의할 사항은 무엇입니까? {#aws-eks}

AWS EKS 환경에 배포할 경우 DataKit-Operator가 작동하지 않을 수 있으므로 보안 그룹에서 `9543` 포트를 개방해야 합니다.

### 문제 해결 가이드 {#debug}

| 문제 | 가능한 원인 | 해결 방법 |
|--- |--- |---|
| 주입이 적용되지 않음 | Webhook이 올바르게 구성되지 않음 | `MutatingAdmissionWebhook` 및 `ValidatingAdmissionWebhook` 확인 |
| 이미지 가져오기 실패 | 이미지 주소 또는 권한 문제 | 이미지 주소를 검증하고 이미지 레지스트리 액세스 권한 확인 |
| 포트에 연결할 수 없음 | 네트워크 또는 보안 그룹 구성 | `9543` 포트를 개방하고 네트워크 정책 확인 |
