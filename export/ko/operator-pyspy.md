# 레거시 Python Profiler 주입

이 기능은 `datakit-profiler` 사이드카에서 py-spy를 실행하는 레거시 배포 호환용 주입 방식이며 CPython만 지원합니다. 새로 배포할 때는 [Flameshot](operator-flameshot.md)을 사용하는 것이 좋습니다.

## Operator 구성 {#prerequisites}

현재 버전에서는 먼저 `admission_inject_v2.profilers`에 일치 규칙을 구성해야 합니다. Annotation만 추가해서는 주입되지 않습니다.

```json
{
    "admission_inject_v2": {
        "profilers": [
            {
                "name": "legacy-python-profiler",
                "language": "python",
                "namespace_selectors": ["^production$"],
                "label_selectors": ["profiling=py-spy"],
                "check_annotation": false,
                "image": "{{.K8sProfilersPySpyImage}}",
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

`check_annotation: true`이면 Pod에 `admission.datakit/python-profiler.version`도 있어야 하며, 이 값은 이미지 태그만 대체합니다. 개별 Pod에서 레거시 Profiler를 비활성화하려면 `admission.datakit/profiler.enabled: "false"`를 설정합니다.

규칙이 일치하면 Operator는 `datakit-profiler` 사이드카, 공유 프로세스 네임스페이스, 작업 디렉터리, `/tmp` 및 `/etc/localtime` 마운트를 추가하고 Pod의 `restartPolicy`를 `Always`로 설정합니다. 사이드카에는 `SYS_PTRACE` 및 `SYS_ADMIN` capability가 추가되므로 사용하기 전에 Pod 보안 정책에서 허용되는지 확인하십시오.

## Deployment 예시 {#pyspy-example}

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: movies-python
  namespace: production
spec:
  replicas: 1
  selector:
    matchLabels:
      app: movies-python
  template:
    metadata:
      labels:
        app: movies-python
        profiling: py-spy
    spec:
      containers:
        - name: app
          image: example/movies-python:1.2.3
```

생성한 후 `kubectl get pod -n production -l app=movies-python -o yaml`을 실행하여 `datakit-profiler`가 있는지 확인합니다. 데이터가 없으면 사이드카 로그, capability, 대상 프로세스의 CPython 여부 및 DataKit Profile 수신 주소를 확인하십시오.
