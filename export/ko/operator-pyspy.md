# DataKit Operator를 통한 Python 프로파일링 주입

## 사전 요구 사항 {#prerequisites}

- 현재는 Python 공식 인터프리터(CPython)만 지원합니다.

[Pod 컨트롤러](https://kubernetes.io/docs/concepts/workloads/controllers/){:target="_blank"} 리소스 구성 파일의
`.spec.template.metadata.annotations` 노드 아래에 다음 annotation을 추가한 후 해당 리소스 구성 파일을 적용하면,
DataKit-Operator가 해당 Pod에 프로파일링을 지원하는 `datakit-profiler` 컨테이너를 자동으로 생성합니다.

> **Annotation 사용 안내**: `check_annotation` 구성이 버전 annotation의 동작에 미치는 영향과 각 annotation에 관한 자세한 설명은 [Annotation 구성 주입](datakit-operator.md#annotation-injection) 및 [`check_annotation` 구성 항목 설명](datakit-operator.md#check-annotation-config)을 참조하십시오.

다음은 "movies-python"이라는 `Deployment` 리소스 구성 파일을 예로 들어 설명합니다.

```yaml hl_lines="17"
apiVersion: apps/v1
kind: Deployment
metadata:
  name: movies-python
  labels:
    app: movies-python
spec:
  replicas: 1
  selector:
    matchLabels:
      app: movies-python
  template:
    metadata:
      name: movies-python
      labels:
        app: movies-python
      annotations:
        admission.datakit/python-profiler.version: {{.K8sProfilersPySpyVersion}} # <-- add annotation here
    spec:
      containers:
        - name: movies-python
          image: zhangyicloud/movies-python:1.2.3
          imagePullPolicy: Always
          command:
            - "gunicorn"
            - "-w"
            - "4"
            - "--bind"
            - "0.0.0.0:8080"
            - "app:app"
```

리소스 구성을 적용하고 적용 여부를 확인합니다.

```shell
$ kubectl apply -f deployment-movies-python.yaml

$ kubectl get pods | grep movies-python
movies-python-78b6cf55f-ptzxf   2/2     Running   0          64s


$ kubectl describe pod movies-python-78b6cf55f-ptzxf | grep datakit-profiler
      /app/datakit-profiler from datakit-profiler-volume (rw)
  datakit-profiler:
      /app/datakit-profiler from datakit-profiler-volume (rw)
  datakit-profiler-volume:
  Normal  Created    98s   kubelet            Created container datakit-profiler
  Normal  Started    97s   kubelet            Started container datakit-profiler
```

몇 분 정도 기다리면 <<<custom_key.brand_name>>> 콘솔의 [애플리케이션 성능 모니터링(APM)-프로파일링](https://console.<<<custom_key.brand_main_domain>>>/tracing/profile){:target="_blank"} 페이지에서 애플리케이션 성능 데이터를 확인할 수 있습니다.

<!-- markdownlint-disable MD046 -->
???+ note

    - 기본적으로 `ps -e -o pid,cmd --no-headers | grep -v grep | grep "python" | head -n 20` 명령을 사용하여 컨테이너의 `Python` 프로세스를 찾습니다. 성능을 고려하여 최대 20개 프로세스의 데이터만 수집합니다.

    - `datakit-operator.yaml` 구성 파일에서 ConfigMap `datakit-operator-config` 아래의 환경 변수를 수정하여 프로파일링 동작을 구성할 수 있습니다.

    | 환경 변수              | 설명                                                                                                                                               | 기본값                        |
    | ----                  | --                                                                                                                                                 | -----                         |
    | `DK_PROFILE_SCHEDULE` | 프로파일링 실행 일정입니다. Linux [Crontab](https://man7.org/linux/man-pages/man5/crontab.5.html){:target="_blank"}과 동일한 구문을 사용합니다. 예: `*/10 * * * *` | `0 * * * *`(매시간 한 번 실행) |
    | `DK_PROFILE_DURATION` | 각 프로파일링의 지속 시간이며 단위는 초입니다.                                                                                                                  | 240(4분)                 |


    - 데이터가 표시되지 않으면 `datakit-profiler` 컨테이너에 접속하여 관련 로그를 확인하고 문제를 진단할 수 있습니다.

    ```shell
    $ kubectl exec -it movies-python-78b6cf55f-ptzxf -c datakit-profiler -- bash
    $ tail -n 2000 log/main.log
    ```
<!-- markdownlint-enable MD046 -->