# Operator 업데이트 내역

## 1.8.9(2026-06-23) {#cl-1.8.9}

- Cluster API Pod 조회 성능을 최적화하고 eBPF용 간소화 뷰를 지원하여 대규모 클러스터에서 응답 본문 크기와 파싱 오버헤드를 줄였습니다（#94）

## 1.8.8(2026-05-28) {#cl-1.8.8}

- ddtrace 주입 시 `check_annotation` 설정 항목이 예상대로 적용되지 않을 수 있던 문제를 수정했습니다（#91）

## 1.8.7(2026-05-14) {#cl-1.8.7}

- Helm Chart와 YAML의 이미지 태그를 업데이트했습니다

## 1.8.6(2026-05-12) {#cl-1.8.6}

- Node.js ddtrace agent 주입을 지원합니다（#89）

## 1.8.5(2026-05-09) {#cl-1.8.5}

- 일부 리소스 주입 시 기존 Pod 필드가 덮어써지거나 누락되어 Pod가 정상적으로 실행되지 않던 문제를 수정했습니다（#88）

## 1.8.4(2026-04-03) {#cl-1.8.4}

- 리소스 주입 시 initContainer의 RestartPolicy 설정이 누락될 수 있던 문제를 수정했습니다（#86）

## 1.8.3(2026-03-20) {#cl-1.8.3}

- PHP ddtrace agent 주입을 지원합니다（#82）

## 1.8.2(2026-03-10) {#cl-1.8.2}

- 일부 상황에서 `/logging/configs` API가 비정상적으로 응답하던 문제를 수정했습니다（#85）
- Operator 실행 상태를 쉽게 점검할 수 있도록 일부 로그 출력을 최적화했습니다

## 1.8.1(2026-02-11) {#cl-1.8.1}

- `resourceFieldRef` 형식의 환경 변수 주입을 지원하며, 이제 limits.cpu, limits.memory, requests.cpu, requests.memory를 비롯한 컨테이너 리소스 제한 및 요청 값을 참조할 수 있습니다（#84）
- Python ddtrace agent 주입을 지원합니다（#81）
- 이 클러스터의 Pod 관련 데이터를 가져오는 프록시 API를 제공합니다（#79）

## 1.8.0(2026-01-29) {#cl-1.8.0}

- `admission_inject_v2` 설정 항목에 `check_annotation` 필드를 추가하여 `admission.datakit/java-lib.version` 사용 방식과 호환되도록 했습니다（#81）
- 기존 설정 `admission_inject.profiler`을 통한 Profiler 주입 기능과 호환됩니다（#81）

## 1.7.3(2026-01-08) {#cl-1.7.3}

- ClusterLoggingConfig CRD를 지원하는 `from_beginning_threshold_size` 설정 항목을 추가했습니다（#80）

## 1.7.2(2025-12-18) {#cl-1.7.2}

- logfwd 주입 시 발생할 수 있던 마운트 오류를 수정했습니다（#78）

## 1.7.1(2025-12-17) {#cl-1.7.1}

- 기존 설정과의 호환성 문제를 수정했습니다（#77）
- `admission_inject_v2` 설정 항목의 `namespace_selectors` 및 `label_selectors` 값이 모두 비어 있으면 주입을 수행하지 않습니다（#77）
- flameshot 주입에 Prometheus.io Annotations를 추가할 수 있는 `enable_prometheus_annotations` 설정 항목을 추가했습니다（#74）
- flameshot 포트 설정 방식을 환경 변수 `FLAMESHOT_HTTP_LOCAL_ADDR` 사용 방식에서 `FLAMESHOT_HTTP_LOCAL_PORT` 사용 방식으로 변경했습니다（#75）
- logfwd는 CRD 설정 소스를 사용할 수 있으므로 logfwd 주입 시 `log_configs`을 반드시 설정해야 했던 잘못된 동작을 수정했습니다（#76）

## 1.7.0(2025-12-12) {#cl-1.7.0}

- 새로운 `admission_inject_v2` 설정 항목을 추가하여 기존 `admission_inject`을 대체하고 이전 버전과의 호환성을 유지합니다（#73）
- 기존 Profiler 주입을 대체하는 flameshot 주입을 추가했습니다（#72）
- 선택기 `namespace_selectors` 및 `label_selectors`을 함께 설정할 때 두 선택기의 관계를 “OR”에서 “AND”로 변경했습니다
- Annotation `admission.datakit/logfwd.log_configs` 및 `admission.datakit/logfwd.volume_paths`에 대한 지원을 제거했습니다（v1.6.X 버전에서만 지원）
- Annotation `admission.datakit/java-lib.version`에 대한 지원을 제거했습니다（`admission.datakit/ddtrace.enabled:"false"`을 통해 주입을 비활성화할 수 있음）

## 1.6.1(2025-12-02) {#cl-1.6.1}

- Kubernetes ClusterLoggingConfig CRD의 오류 검사를 개선하여 리소스가 없으면 로그를 출력하고 해당 기능을 중지합니다（#71）

## 1.6.0(2025-11-19) {#cl-1.6.0}

- Kubernetes ClusterLoggingConfig CRD 지원을 추가하고 로그 수집 설정을 반환하는 HTTP API를 추가했습니다（#70）
- 새로운 CRD 로그 수집 설정에 맞게 logfwd 주입 방식을 조정하고 최적화했습니다（#70）

## 1.5.18(2025-07-15) {#cl-1.5.18}

- ddtrace 주입 시 Resources Requests 및 Resources Limits를 직접 설정할 수 있습니다（#68）

## 1.5.17(2025-06-03) {#cl-1.5.17}

- 배포 이미지에서 uos arm64 버전을 지원합니다

## 1.5.16(2025-04-16) {#cl-1.5.16}

- logging 설정 주입의 매칭 순서를 조정했습니다（#64）

## 1.5.15(2025-04-15) {#cl-1.5.15}

- Kubernetes 1.19 버전에 배포할 때 namespace 매칭이 지원되지 않던 문제를 개선했습니다（#63）

## 1.5.14(2025-04-01) {#cl-1.5.14}

- namespace 정규식 작성 방식을 개선하여 이제 all을 매칭하려면 `*`만 작성하면 되며 `.*`을 작성할 필요가 없습니다（#59）

## 1.5.13(2025-03-31) {#cl-1.5.13}

- 정규식으로 namespace와 labels를 매칭하여 DDtrace와 Profiler를 주입할 수 있습니다（#58）
- 정규식으로 namespace와 labels를 매칭하여 logging 설정을 주입할 수 있습니다（#59）

## 1.5.12(2025-02-28) {#cl-1.5.12}

- logfwd와 profiler 주입 시 Resources Requests 및 Resources Limits를 직접 설정할 수 있습니다（#57）

## 1.5.11(2025-02-20) {#cl-1.5.11}

- 대상 Pod에 datakit/logs 어노테이션 설정을 추가하고 디렉터리를 자동으로 마운트할 수 있습니다（#53）
- logfwd sidecar 주입 시 디렉터리 마운트 로직을 최적화하여 이제 기존 마운트를 기본적으로 재사용합니다（#55）

## 1.5.10(2024-12-10) {#cl-1.5.10}

- 앞에 정의된 환경 변수 값을 항상 참조할 수 있도록 DD_TAGS 환경 변수의 주입 순서를 조정했습니다（#51）

## 1.5.9(2024-11-25) {#cl-1.5.9}

- 주입 이미지의 pullPolicy를 Always로 변경했습니다（#50）

## 1.5.8(2024-10-10) {#cl-1.5.8}

- ddtrace Python 주입 로직을 변경하여 더 이상 이미지와 lib를 주입하지 않고 기본 환경 변수만 추가합니다（#35）

## 1.5.7(2024-09-18) {#cl-1.5.7}

- v1.5.5와 v1.5.6에서 DD_TAGS 환경 변수를 잘못 주입하던 문제를 수정했습니다（#48）
- Pod에 Annotations（`admission.datakit/ddtrace.enabled="false"`, `admission.datakit/logfwd.enabled="false"` 및 `admission.datakit/profiler.enabled="false"`）를 추가하여 특정 유형의 주입을 더 세밀하게 비활성화할 수 있습니다（#49）

## 1.5.6(2024-09-13) {#cl-1.5.6}

- profiler volumeMount 주입 로직을 최적화하여 동일한 path가 이미 있으면 더 이상 주입하지 않습니다（#46）

## 1.5.5(2024-09-02) {#cl-1.5.5}

- ddtrace 환경 변수 `DD_TAGS` 주입 로직을 최적화하여 기존 Pod에 `DD_TAGS`이 이미 있으면 이제 무시하지 않고 값을 추가합니다（#45）

## 1.5.4(2024-08-27) {#cl-1.5.4}

- ddtrace 주입 시 리소스를 제거하고 logfwd와 profiler의 리소스를 줄였습니다（#44）

## 1.5.3(2024-04-09) {#cl-1.5.3}

- 다중 컨테이너 환경에서 ddtrace 마운트 파일이 누락되던 문제를 수정했습니다（#42）

## 1.5.2(2024-04-08) {#cl-1.5.2}

- labelSelector에 따른 ddtrace 일괄 주입을 지원합니다（#41）

## 1.5.1(2024-04-07) {#cl-1.5.1}

- 로그에 현재 버전과 빌드 정보를 출력합니다（#41）

## 1.5.0(2024-03-18) {#cl-1.5.0}

- Pod에 Annotation `admission.datakit/enabled="false"`을 추가하여 ddtrace, logfwd 및 profiler 주입을 포함한 모든 주입을 비활성화할 수 있습니다（#39）
- 지정된 namespace에 ddtrace를 일괄 주입할 수 있습니다（#38）
- logfwd 주입 시 volume을 재사용하도록 선택하여 같은 경로가 여러 번 마운트되어 발생하는 오류를 방지할 수 있습니다（#34）

## 1.4.3(2023-12-21) {#cl-1.4.3}

- logfwd 주입 시 logfiles에 와일드카드 경로를 지정하면 mount 오류가 발생하던 문제를 수정했습니다（#31）
- logfwd 주입 시 Pod에 컨테이너가 2개 이상이면 주입에 실패하고 기존 Pod 시작에 영향을 주던 문제를 수정했습니다（#32）
- 로그 출력 한 곳을 최적화했습니다

## 1.4.2(2023-09-18) {#cl-1.4.2}

- 주입된 환경 변수의 순서가 일치하지 않던 문제를 수정했습니다 (#28)

## 1.4.1(2023-09-15) {#cl-1.4.1}

- logfwd에 환경 변수를 주입할 수 있습니다 (#27)

## 1.4.0(2023-09-13) {#cl-1.4.0}

- Kubernetes DownloadAPI FieldRef 방식으로 환경 변수를 설정할 수 있습니다 (#26)
- ddtrace에 DD_TAGS를 기본으로 추가합니다 (#26)

## 1.3.1(2023-08-24) {#cl-1.3.1}

- 기본 Profiler 이미지를 업데이트했습니다

## 1.3.0(2023-07-17) {#cl-1.3.0}

- admission 주입의 최소 단위를 Pod로 변경했으며, yaml을 최신 버전 또는 datakit-operator-v1.3.0.yaml로 업데이트해야 합니다 (#22)
- profiler 주입을 지원합니다 (#5)
- 주입되는 sidecar에 Resource Limit을 추가했습니다 (#20)

## 1.2.1(2023-06-28) {#cl-1.2.1}

- 기본 이미지 저장소를 변경했습니다 (#19)
- Helm 구조를 업데이트했습니다

## 1.2.0(2023-06-13) {#cl-1.2.0}

- Datakit Operator를 JSON 방식으로 설정할 수 있으며, 기존 환경 변수 방식도 계속 호환됩니다 (#19)

## 1.0.5(2023-05-11) {#cl-1.0.5}

- 새로운 ping API를 추가했습니다 (#18)
- Datakit 리더 선출 전용 API를 추가하여 Datakit 수집기에 작업을 분배할 수 있습니다 (#15)

## 1.0.4(2023-04-10) {#cl-1.0.4}

- Kubernetes Admission 방식의 logfwd 프로그램 주입을 지원합니다 (#12)
- ddtrace agent 주입 시 `DD_AGENT_HOST` 및 `DD_TRACE_AGENT_PORT` 환경 변수를 기본으로 추가합니다 (#14)
- Admission이 지원하는 resources 목록을 변경하고 네이티브 Pod에 대한 지원을 제거했습니다 (#12)
- 코드 구조를 변경하고 ddtrace와 logfwd의 단위 테스트를 보완했습니다
- datakit.yaml 구조를 최적화했습니다
- docs 디렉터리와 문서를 제거하고 README에 새로운 문서 링크를 제공합니다

## 1.0.3(2023-03-27) {#cl-1.0.3}

- 영문 문서를 추가했습니다
- 환경 변수 방식으로 dd-agent 이미지 주소를 설정할 수 있습니다 (#9)
- 여러 환경 변수의 이름을 최적화했습니다
- datakit.yaml을 변경하여 기본적으로 webhook 오류를 무시하도록 했습니다 (#11)

## 1.0.2(2023-03-09) {#cl-1.0.2}

- CHANGELOG를 추가했습니다 (#9)
- 인증서 만료로 인해 접근에 실패하던 문제를 수정하고 자체 서명 인증서를 다시 생성하여 만료 기간을 연장했습니다 (#8)
- 이미지 배포 시 발생하던 세부 오류 하나를 수정했습니다 (#10)
- yaml 설치 방식을 변경하여 yaml에서 namespace를 생성하지 않으며, 문서에 관련 설명을 보완했습니다 (#7)

## 1.0.1(2022-12-28) {#cl-1.0.1}

- Makefile, Dockerfile 및 CI 설정을 추가했습니다 (#2)
- Kubernetes Admission 방식의 ddtrace 파일 및 환경 변수 주입을 지원합니다 (#1)
