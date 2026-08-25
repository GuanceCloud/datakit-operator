# DataKit Operator에 async-profiler 주입

## 사전 요구 사항 {#async-profiler-prerequisites}

- 클러스터에 [DataKit](https://docs.<<<custom_key.brand_main_domain>>>/datakit/datakit-daemonset-deploy/){:target="_blank"}이 설치되어 있어야 합니다.
- profile 수집기를 [활성화](https://docs.<<<custom_key.brand_main_domain>>>/datakit/datakit-daemonset-deploy/#using-k8-env){:target="_blank"}해야 합니다.
- Linux 커널 매개변수 [kernel.perf_event_paranoid](https://www.kernel.org/doc/Documentation/sysctl/kernel.txt){:target="_blank"} 값을 2 이하로 설정해야 합니다.

<!-- markdownlint-disable MD046 -->
???+ note

    `async-profiler`에서는 [`perf_events`](https://perf.wiki.kernel.org/index.php/Main_Page){:target="_blank"} 도구를 사용하여 Linux 커널 호출 스택을 캡처합니다. 비특권 프로세스는 관련 커널 설정에 의존하므로 다음 명령으로 커널 매개변수를 변경할 수 있습니다.
    ```shell
    $ sudo sysctl kernel.perf_event_paranoid=1
    $ sudo sysctl kernel.kptr_restrict=0
    # 또는
    $ sudo sh -c 'echo 1 >/proc/sys/kernel/perf_event_paranoid'
    $ sudo sh -c 'echo 0 >/proc/sys/kernel/kptr_restrict'
    ```
<!-- markdownlint-enable MD046 -->
## 주입 구성 {#annotation-injection}

[Pod 컨트롤러](https://kubernetes.io/docs/concepts/workloads/controllers/){:target="_blank"} 리소스 구성 파일의
`.spec.template.metadata.annotations` 노드 아래에 다음 annotation을 추가한 후 해당 리소스 구성 파일을 적용합니다.
DataKit-Operator는 해당 Pod에 이름이 `datakit-profiler`인 컨테이너를 자동으로 생성하여 프로파일링을 지원합니다.

> **Annotation 사용 안내**: `check_annotation` 설정이 버전 annotation의 동작에 미치는 영향과 각 annotation에 대한 자세한 설명은 [Annotation 주입 구성](datakit-operator.md#annotation-injection) 및 [`check_annotation` 구성 항목 설명](datakit-operator.md#check-annotation-config)을 참조합니다.

다음 Deployment 리소스 구성 파일을 예로 들어 설명합니다.

```yaml hl_lines="17"
kind: Deployment
metadata:
  name: movies-java
  labels:
    app: movies-java
spec:
  replicas: 1
  selector:
    matchLabels:
      app: movies-java
  template:
    metadata:
      name: movies-java
      labels:
        app: movies-java
      annotations:
        admission.datakit/java-profiler.version: "{{.K8sProfilersAsyncProfileVersion}}" # <-- add annotation here
    spec:
      containers:
        - name: movies-java
          image: your/app:v1.2.3
          imagePullPolicy: IfNotPresent
          securityContext:
            seccompProfile:
              type: Unconfined
          env:
            - name: JAVA_OPTS
              value: ""

      restartPolicy: Always
```

구성 파일을 적용하고 적용 여부를 확인합니다.

```shell
$ kubectl apply -f deployment-movies-java.yaml

$ kubectl get pods | grep movies-java
movies-java-784f4bb8c7-59g6s   2/2     Running   0          47s

$ kubectl describe pod movies-java-784f4bb8c7-59g6s | grep datakit-profiler
      /app/datakit-profiler from datakit-profiler-volume (rw)
  datakit-profiler:
      /app/datakit-profiler from datakit-profiler-volume (rw)
  datakit-profiler-volume:
  Normal  Created    12m   kubelet            Created container datakit-profiler
  Normal  Started    12m   kubelet            Started container datakit-profiler
```

몇 분 후 <<<custom_key.brand_name>>> 콘솔의 [애플리케이션 성능 모니터링(APM)-프로파일링](https://console.<<<custom_key.brand_main_domain>>>/tracing/profile){:target="_blank"} 페이지에서 애플리케이션 성능 데이터를 확인할 수 있습니다.

<!-- markdownlint-disable MD046 -->
???+ note

    - 기본적으로 `jps -q -J-XX:+PerfDisableSharedMem | head -n 20` 명령을 사용하여 컨테이너의 JVM 프로세스를 찾습니다. 성능을 고려하여 최대 20개 프로세스의 데이터만 수집합니다.

    - `datakit-operator.yaml` 구성 파일에서 `datakit-operator-config` 아래의 환경 변수를 수정하여 프로파일링 동작을 구성할 수 있습니다.


    | 환경 변수              | 설명                                                                                                                                               | 기본값                        |
    | ----                  | --                                                                                                                                                 | -----                         |
    | `DK_PROFILE_SCHEDULE` | 프로파일링 실행 일정입니다. Linux [Crontab](https://man7.org/linux/man-pages/man5/crontab.5.html){:target="_blank"}과 동일한 구문을 사용합니다(예: `*/10 * * * *`). | `0 * * * *`(시간마다 한 번 실행) |
    | `DK_PROFILE_DURATION` | 각 프로파일링의 지속 시간이며 단위는 초입니다.                                                                                                                  | 240(4분)                 |


    - 데이터가 표시되지 않으면 `datakit-profiler` 컨테이너에 접속하여 관련 로그를 확인하고 문제를 해결할 수 있습니다.

    ```shell
    $ kubectl exec -it movies-java-784f4bb8c7-59g6s -c datakit-profiler -- bash
    $ tail -n 2000 log/main.log
    ```
<!-- markdownlint-enable MD046 -->