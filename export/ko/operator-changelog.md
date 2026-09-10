# Operator 변경 내역

## 1.9.2(2026-09-10) {#cl-1.9.2}

- `image_pull_policy`로 주입 이미지의 가져오기 정책을 설정하는 기능 지원. 값이 없거나 잘못되면 기본값 `Always` 사용 (#110)

## 1.9.1(2026-09-09) {#cl-1.9.1}

- Kubernetes Lease 기반 DataKit 중앙 선출 지원. DataKit 2.12.0 이상과 함께 사용해야 함 (#109)
- Lease 캐시 지연으로 새 Leader의 리스 갱신이 잘못 거부될 수 있는 문제 수정 (#109)
- Pod 캐시 동기화를 백그라운드에서 실행하도록 변경. 캐시가 준비되기 전에는 조회 API가 503을 반환하여 다른 API의 시작을 차단하지 않도록 개선 (#109)
- 브랜치 Pipeline에서 날짜 및 commit 태그가 포함된 RC 이미지를 수동으로 릴리스하는 기능 지원 (#108)

## 1.9.0(2026-08-26) {#cl-1.9.0}

- Java, Python 및 Node.js 애플리케이션에 OpenTelemetry 자동 계측 주입 지원 (#98, #101, #102)
- 호환되지 않는 설정이 애플리케이션 Pod에 영향을 주지 않도록 계측 주입의 안전성과 설정 오류 허용성 개선 (#87, #105)
- Admission Webhook 재호출 시 DDTrace Java가 적용되지 않을 수 있는 문제 수정 (#100)

## 1.8.10(2026-07-06) {#cl-1.8.10}

- `secretKeyRef`로 Kubernetes Secret을 참조하는 환경 변수 주입 지원 (#95)

## 1.8.9(2026-06-23) {#cl-1.8.9}

- Cluster API Pod 조회 성능을 개선하고 eBPF용 간소화 뷰를 지원하여 대규모 클러스터의 응답 본문 크기와 파싱 오버헤드 감소 (#94)

## 1.8.8(2026-05-28) {#cl-1.8.8}

- DDTrace 주입 시 `check_annotation` 구성 항목이 예상대로 적용되지 않을 수 있는 문제 수정 (#91)

## 1.8.7(2026-05-14) {#cl-1.8.7}

- Helm Chart와 YAML의 이미지 tag 업데이트

## 1.8.6(2026-05-12) {#cl-1.8.6}

- Node.js DDTrace Agent 주입 지원 (#89)

## 1.8.5(2026-05-09) {#cl-1.8.5}

- 일부 리소스 주입 시 기존 Pod 필드가 덮어써지거나 누락되어 Pod가 정상적으로 실행되지 않는 문제 수정 (#88)

## 1.8.4(2026-04-03) {#cl-1.8.4}

- 리소스 주입 시 initContainer의 RestartPolicy 구성이 누락될 수 있는 문제 수정 (#86)

## 1.8.3(2026-03-20) {#cl-1.8.3}

- PHP DDTrace Agent 주입 지원 (#82)

## 1.8.2(2026-03-10) {#cl-1.8.2}

- 일부 상황에서 `/logging/configs` API가 비정상적으로 응답하는 문제 수정 (#85)
- Operator 실행 상태를 쉽게 진단할 수 있도록 일부 로그 출력 개선

## 1.8.1(2026-02-11) {#cl-1.8.1}

- `resourceFieldRef` 형식의 환경 변수 주입을 지원하여 limits.cpu, limits.memory, requests.cpu 및 requests.memory를 비롯한 컨테이너 리소스 limit과 request 값 참조 가능 (#84)
- Python DDTrace Agent 주입 지원 (#82)
- 클러스터 내 Pod 관련 데이터를 가져오는 프록시 API 제공 (#79)

## 1.8.0(2026-01-29) {#cl-1.8.0}

- `admission_inject_v2` 구성 항목에 `check_annotation` 필드를 추가하여 `admission.datakit/java-lib.version` 사용 방식과 호환 (#81)
- 레거시 구성 `admission_inject.profiler`의 Profiler 주입 기능과 호환 (#81)

## 1.7.3(2026-01-08) {#cl-1.7.3}

- ClusterLoggingConfig CRD용 `from_beginning_threshold_size` 구성 항목 추가 (#80)

## 1.7.2(2025-12-18) {#cl-1.7.2}

- logfwd 주입 시 발생할 수 있는 마운트 오류 수정 (#78)

## 1.7.1(2025-12-17) {#cl-1.7.1}

- 레거시 구성과의 호환성 문제 수정 (#77)
- `admission_inject_v2` 구성 항목의 `namespace_selectors`와 `label_selectors`가 모두 비어 있으면 주입하지 않도록 변경 (#77)
- Flameshot 주입에 Prometheus.io Annotations를 추가할 수 있는 `enable_prometheus_annotations` 구성 항목 추가 (#74)
- Flameshot 포트 구성 방식을 환경 변수 `FLAMESHOT_HTTP_LOCAL_ADDR`에서 `FLAMESHOT_HTTP_LOCAL_PORT`로 변경 (#75)
- logfwd가 CRD 구성 소스를 사용할 수 있는데도 주입 시 `log_configs`를 필수로 요구하던 문제 수정 (#76)

## 1.7.0(2025-12-12) {#cl-1.7.0}

- `admission_inject_v2` 구성 항목을 추가하여 기존 `admission_inject`를 대체하고 하위 호환성 유지 (#73)
- 기존 Profiler 주입을 대체하는 Flameshot 주입 추가 (#72)
- `namespace_selectors`와 `label_selectors`를 함께 구성할 때 두 selector의 관계를 OR에서 AND로 변경
- Annotation `admission.datakit/logfwd.log_configs` 및 `admission.datakit/logfwd.volume_paths` 지원 제거(v1.6.X에서만 지원)
- Annotation `admission.datakit/java-lib.version` 지원 제거(`admission.datakit/ddtrace.enabled:"false"`로 주입 비활성화 가능)

## 1.6.1(2025-12-02) {#cl-1.6.1}

- Kubernetes ClusterLoggingConfig CRD 오류 검사를 개선하여 리소스가 없으면 로그를 출력하고 해당 기능을 중지하도록 변경 (#71)

## 1.6.0(2025-11-19) {#cl-1.6.0}

- Kubernetes ClusterLoggingConfig CRD 지원 및 로그 수집 구성을 반환하는 HTTP API 추가 (#70)
- 새로운 CRD 로그 수집 구성에 맞게 logfwd 주입 방식 조정 및 개선 (#70)

## 1.5.18(2025-07-15) {#cl-1.5.18}

- DDTrace 주입 시 Resources Requests 및 Resources Limits를 직접 구성하는 기능 지원 (#68)

## 1.5.17(2025-06-03) {#cl-1.5.17}

- 릴리스 이미지에서 UOS arm64 버전 지원

## 1.5.16(2025-04-16) {#cl-1.5.16}

- Logging 구성 주입의 일치 순서 조정 (#64)

## 1.5.15(2025-04-15) {#cl-1.5.15}

- Kubernetes 1.19에 배포할 때 Namespace 일치가 지원되지 않는 문제 개선 (#63)

## 1.5.14(2025-04-01) {#cl-1.5.14}

- Namespace 정규식 작성 방식을 개선하여 전체와 일치시킬 때 `*`만 사용하고 `.*`는 사용하지 않도록 변경 (#59)

## 1.5.13(2025-03-31) {#cl-1.5.13}

- 정규식으로 Namespace와 Label을 일치시켜 DDTrace 및 Profiler를 주입하는 기능 지원 (#58)
- 정규식으로 Namespace와 Label을 일치시켜 Logging 구성을 주입하는 기능 지원 (#59)

## 1.5.12(2025-02-28) {#cl-1.5.12}

- logfwd 및 Profiler 주입 시 Resources Requests 및 Resources Limits를 직접 구성하는 기능 지원 (#57)

## 1.5.11(2025-02-20) {#cl-1.5.11}

- 대상 Pod에 datakit/logs Annotation 구성을 추가하고 디렉터리를 자동으로 마운트하는 기능 지원 (#53)
- logfwd 사이드카의 디렉터리 마운트 로직을 개선하여 기존 마운트를 기본적으로 재사용 (#55)

## 1.5.10(2024-12-10) {#cl-1.5.10}

- 앞에 정의된 환경 변수 값을 항상 참조할 수 있도록 환경 변수 DD_TAGS의 주입 순서 조정 (#51)

## 1.5.9(2024-11-25) {#cl-1.5.9}

- 주입 이미지의 pullPolicy를 Always로 변경 (#50)

## 1.5.8(2024-10-10) {#cl-1.5.8}

- DDTrace Python 주입 로직을 변경하여 이미지와 lib는 더 이상 주입하지 않고 기본 환경 변수만 추가 (#35)

## 1.5.7(2024-09-18) {#cl-1.5.7}

- v1.5.5 및 v1.5.6에서 환경 변수 DD_TAGS를 잘못 주입하는 문제 수정 (#48)
- Pod에 Annotations(`admission.datakit/ddtrace.enabled="false"`, `admission.datakit/logfwd.enabled="false"` 및 `admission.datakit/profiler.enabled="false"`)를 추가하여 특정 유형의 주입을 세밀하게 비활성화하는 기능 지원 (#49)

## 1.5.6(2024-09-13) {#cl-1.5.6}

- Profiler volumeMount 주입 로직을 개선하여 동일한 path가 이미 있으면 주입하지 않도록 변경 (#46)

## 1.5.5(2024-09-02) {#cl-1.5.5}

- DDTrace 환경 변수 `DD_TAGS` 주입 로직을 개선하여 기존 Pod에 `DD_TAGS`가 있으면 무시하지 않고 추가하도록 변경 (#45)

## 1.5.4(2024-08-27) {#cl-1.5.4}

- DDTrace 주입에서 resource를 제거하고 logfwd 및 Profiler의 resource 축소 (#44)

## 1.5.3(2024-04-09) {#cl-1.5.3}

- 다중 컨테이너 환경에서 DDTrace 마운트 파일이 누락되는 문제 수정 (#42)

## 1.5.2(2024-04-08) {#cl-1.5.2}

- labelSelector에 따른 DDTrace 일괄 주입 지원 (#41)

## 1.5.1(2024-04-07) {#cl-1.5.1}

- 로그에 현재 버전과 빌드 정보 출력 (#41)

## 1.5.0(2024-03-18) {#cl-1.5.0}

- Pod에 Annotation `admission.datakit/enabled="false"`를 추가하여 DDTrace, logfwd 및 Profiler를 포함한 모든 주입을 비활성화하는 기능 지원 (#39)
- 지정한 Namespace에 DDTrace를 일괄 주입하는 기능 지원 (#38)
- logfwd 주입 시 볼륨을 재사용할 수 있도록 하여 동일한 경로의 중복 마운트 오류 방지 (#34)

## 1.4.3(2023-12-21) {#cl-1.4.3}

- logfwd 주입 시 logfiles에 와일드카드 경로를 작성하면 마운트 오류가 발생하는 문제 수정 (#31)
- logfwd 주입 시 Pod에 컨테이너가 2개 이상이면 주입에 실패하고 기존 Pod 시작에 영향을 주는 문제 수정 (#32)
- 로그 출력 한 곳 개선

## 1.4.2(2023-09-18) {#cl-1.4.2}

- 주입된 환경 변수의 순서가 일치하지 않는 문제 수정 (#28)

## 1.4.1(2023-09-15) {#cl-1.4.1}

- logfwd 환경 변수 주입 지원 (#27)

## 1.4.0(2023-09-13) {#cl-1.4.0}

- Kubernetes Downward API FieldRef 방식의 환경 변수 구성 지원 (#26)
- DDTrace에 DD_TAGS 기본 추가 (#26)

## 1.3.1(2023-08-24) {#cl-1.3.1}

- 기본 Profiler 이미지 업데이트

## 1.3.0(2023-07-17) {#cl-1.3.0}

- Admission 주입의 최소 단위를 Pod로 변경. YAML을 최신 버전 또는 datakit-operator-v1.3.0.yaml로 업데이트해야 함 (#22)
- Profiler 주입 지원 (#5)
- 주입되는 사이드카에 Resource Limit 추가 (#20)

## 1.2.1(2023-06-28) {#cl-1.2.1}

- 기본 이미지 레지스트리 변경 (#19)
- Helm 구조 업데이트

## 1.2.0(2023-06-13) {#cl-1.2.0}

- DataKit Operator의 JSON 구성 방식을 지원하며 기존 환경 변수 방식과 호환 유지 (#19)

## 1.0.5(2023-05-11) {#cl-1.0.5}

- 새 ping API 추가 (#18)
- DataKit 리더 선출 전용 API를 추가하여 DataKit 수집기에 작업 분배 (#15)

## 1.0.4(2023-04-10) {#cl-1.0.4}

- Kubernetes Admission 방식의 logfwd 프로그램 주입 지원 (#12)
- DDTrace Agent 주입 시 `DD_AGENT_HOST` 및 `DD_TRACE_AGENT_PORT` 환경 변수를 기본으로 추가 (#14)
- Admission이 지원하는 resources 목록을 변경하고 네이티브 Pod 지원 제거 (#12)
- 코드 구조를 변경하고 DDTrace 및 logfwd 단위 테스트 보완
- datakit.yaml 구조 개선
- docs 디렉터리와 문서를 제거하고 README에 새 문서 링크 추가

## 1.0.3(2023-03-27) {#cl-1.0.3}

- 영어 문서 추가
- 환경 변수 방식의 dd-agent 이미지 주소 구성 지원 (#9)
- 여러 환경 변수 이름 개선
- datakit.yaml을 변경하여 기본적으로 webhook 오류 무시 (#11)

## 1.0.2(2023-03-09) {#cl-1.0.2}

- CHANGELOG 추가 (#9)
- 인증서 만료로 접근에 실패하는 문제를 수정하고 자체 서명 인증서를 다시 생성하여 만료 기간 연장 (#8)
- 이미지 릴리스 시 발생하는 세부 오류 수정 (#10)
- YAML 설치 방식을 변경하여 YAML에서 Namespace를 생성하지 않고 문서에 관련 설명 보완 (#7)

## 1.0.1(2022-12-28) {#cl-1.0.1}

- Makefile, Dockerfile 및 CI 구성 추가 (#2)
- Kubernetes Admission 방식의 DDTrace 파일 및 환경 변수 주입 지원 (#1)
