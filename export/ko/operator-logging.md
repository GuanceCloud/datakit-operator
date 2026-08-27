# DataKit Operator 로그 수집 구성 주입

DataKit Operator는 새 Pod에 `datakit/logs` Annotation을 추가하고 파일 로그 경로에 따라 EmptyDir 볼륨을 생성하거나 재사용할 수 있습니다. 따라서 각 Deployment에서 로그 Annotation과 디렉터리 마운트를 반복해서 관리할 필요가 없습니다.

이 기능은 DataKit이 Kubernetes Pod의 파일 로그를 직접 수집하는 경우에 적합합니다. 사이드카를 통해 로그를 DataKit으로 전달하려면 [logfwd 주입](operator-logfwd.md)을 사용하십시오.

## Operator 구성 {#logging-config}

`admission_mutate.loggings`에 규칙을 추가합니다.

```json
{
    "admission_mutate": {
        "loggings": [
            {
                "namespace_selectors": ["^middleware$"],
                "label_selectors": ["app=logging"],
                "config": "[{\"disable\":false,\"type\":\"file\",\"path\":\"/var/log/app/*.log\",\"source\":\"logging-demo\"}]"
            }
        ]
    }
}
```

| 필드 | 설명 |
| --- | --- |
| `namespace_selectors` | Namespace 정규식 배열. 일치 가능한 항목을 하나 이상 구성해야 합니다. |
| `label_selectors` | Pod Label Selector 배열. 일치 가능한 항목을 하나 이상 구성해야 합니다. |
| `config` | `datakit/logs`에 기록할 JSON 배열 문자열 |

Namespace와 Label 두 조건이 모두 일치해야 합니다. 같은 조건의 여러 selector는 OR로 평가하며, 여러 규칙은 구성 순서에 따라 첫 번째로 일치하는 항목을 사용합니다.

`config`는 유효한 JSON이어야 합니다. Operator는 비활성화되지 않은 `type: "file"` 구성을 파싱하고 `path`에서 디렉터리를 추출하여 마운트합니다. `stdout` 구성은 Annotation에만 기록하며 새 볼륨을 추가하지 않습니다.

## 주입 결과 {#logging-injection-result}

위 예시와 일치하는 Pod에는 다음 항목이 추가됩니다.

```yaml
metadata:
  annotations:
    datakit/logs: '[{"disable":false,"type":"file","path":"/var/log/app/*.log","source":"logging-demo"}]'
spec:
  containers:
    - name: app
      volumeMounts:
        - name: datakit-logs-volume-0
          mountPath: /var/log/app
  volumes:
    - name: datakit-logs-volume-0
      emptyDir: {}
```

Operator는 디렉터리를 모든 일반 애플리케이션 컨테이너에 마운트합니다.

- 해당 경로에 EmptyDir가 이미 마운트되어 있으면 기존 볼륨을 재사용합니다.
- 해당 마운트가 없으면 새 EmptyDir를 생성합니다.
- 해당 경로에 다른 유형의 볼륨이 사용 중이면 warning을 기록하고 그 경로를 건너뜁니다.
- Pod에 `datakit/logs` Annotation이 이미 있으면 사용자 구성을 유지하고 Annotation이나 볼륨을 변경하지 않습니다.

## Deployment 예시 {#logging-example}

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: logging-demo
  namespace: middleware
spec:
  replicas: 1
  selector:
    matchLabels:
      app: logging
  template:
    metadata:
      labels:
        app: logging
    spec:
      containers:
        - name: app
          image: nginx:1.25
```

Pod를 생성한 후 다음을 확인합니다.

```shell
kubectl -n middleware get pod -l app=logging -o yaml
kubectl logs -n datakit deployment/datakit-operator
```

최종 Pod에는 `datakit/logs` Annotation과 `/var/log/app`에 해당하는 EmptyDir 및 마운트가 있어야 합니다. 규칙을 변경한 후에는 Pod를 다시 생성해야 적용됩니다.
