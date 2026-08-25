# 레거시 Java Profiler 주입

이 기능은 `datakit-profiler` 사이드카에서 async-profiler를 실행하는 레거시 배포 호환용 주입 방식입니다. 새로 배포할 때는 [Flameshot](operator-flameshot.md)을 사용하는 것이 좋습니다.

## 사전 요구 사항 {#async-profiler-prerequisites}

- DataKit에서 Profile 수집기가 활성화되어 있어야 합니다.
- 노드에서 `perf_events`를 허용해야 하며, 일반적으로 `kernel.perf_event_paranoid` 값이 `2` 이하여야 합니다.
- Pod 보안 정책에서 사이드카에 `SYS_PTRACE` 및 `SYS_ADMIN` capability를 추가할 수 있어야 합니다.

## Operator 구성 {#annotation-injection}

현재 버전에서는 먼저 `admission_inject_v2.profilers`에 일치 규칙을 구성해야 합니다. Annotation만 추가해서는 주입되지 않습니다.

```json
{
    "admission_inject_v2": {
        "profilers": [
            {
                "name": "legacy-java-profiler",
                "language": "java",
                "namespace_selectors": ["^production$"],
                "label_selectors": ["profiling=async-profiler"],
                "check_annotation": false,
                "image": "{{.K8sProfilersAsyncProfileImage}}",
                "envs": {
                    "DK_AGENT_HOST": "datakit-service.datakit.svc.cluster.local",
                    "DK_AGENT_PORT": "9529",
                    "DK_PROFILE_DURATION": "240",
                    "DK_PROFILE_SCHEDULE": "0 * * * *"
                }
            }
        ]
    }
}
```

`check_annotation: true`이면 Pod에 `admission.datakit/java-profiler.version`도 있어야 하며, 이 값은 이미지 태그만 대체합니다. 개별 Pod에서 레거시 Profiler를 비활성화하려면 `admission.datakit/profiler.enabled: "false"`를 설정합니다.

규칙이 일치하면 Operator는 `datakit-profiler` 사이드카, 공유 프로세스 네임스페이스, 작업 디렉터리, `/tmp` 및 `/etc/localtime` 마운트를 추가하고 Pod의 `restartPolicy`를 `Always`로 설정합니다. 운영 환경에 적용하기 전에 이러한 변경 사항이 Pod 보안 정책에 부합하는지 확인하십시오.

## Deployment 예시 {#async-profiler-example}

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: movies-java
  namespace: production
spec:
  replicas: 1
  selector:
    matchLabels:
      app: movies-java
  template:
    metadata:
      labels:
        app: movies-java
        profiling: async-profiler
    spec:
      containers:
        - name: app
          image: example/movies-java:1.2.3
          securityContext:
            seccompProfile:
              type: Unconfined
```

생성한 후 `kubectl get pod -n production -l app=movies-java -o yaml`을 실행하여 `datakit-profiler`가 있는지 확인합니다. 데이터가 없으면 사이드카 로그, 커널 매개변수, capability 및 DataKit Profile 수신 주소를 확인하십시오.
