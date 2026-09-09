# Operator Changelog

## 1.9.1(2026-09-09) {#cl-1.9.1}

- Added Kubernetes Lease-based central election for DataKit, requiring DataKit 2.12.0 or later (#109)
- Fixed an issue where a delayed Lease cache could incorrectly reject a new Leader's lease renewal (#109)
- Moved Pod cache synchronization to the background. Query endpoints return 503 until the cache is ready, allowing other endpoints to start without waiting (#109)
- Added manual RC image releases from branch pipelines with date and commit tags (#108)

## 1.9.0(2026-08-26) {#cl-1.9.0}

- Added OpenTelemetry automatic instrumentation injection for Java, Python, and Node.js applications (#98, #101, #102)
- Improved instrumentation injection safety and configuration fault tolerance to prevent incompatible configurations from affecting application Pods (#87, #105)
- Fixed an issue where DDTrace Java might not take effect when the Admission Webhook was invoked again (#100)

## 1.8.10(2026-07-06) {#cl-1.8.10}

- Added support for injecting environment variables that reference Kubernetes Secrets through `secretKeyRef` (#95)

## 1.8.9(2026-06-23) {#cl-1.8.9}

- Improved Cluster API Pod query performance and added a reduced view for eBPF, decreasing response size and parsing overhead in large clusters (#94)

## 1.8.8(2026-05-28) {#cl-1.8.8}

- Fixed an issue where `check_annotation` might not take effect as expected during ddtrace injection (#91)

## 1.8.7(2026-05-14) {#cl-1.8.7}

- Updated image tags in the Helm Chart and YAML

## 1.8.6(2026-05-12) {#cl-1.8.6}

- Added support for injecting the Node.js ddtrace agent (#89)

## 1.8.5(2026-05-09) {#cl-1.8.5}

- Fixed an issue where fields from the original Pod could be overwritten or lost in some resource injection scenarios, preventing the Pod from running correctly (#88)

## 1.8.4(2026-04-03) {#cl-1.8.4}

- Fixed an issue where the RestartPolicy of an initContainer could be discarded during resource injection (#86)

## 1.8.3(2026-03-20) {#cl-1.8.3}

- Added support for injecting the PHP ddtrace agent (#82)

## 1.8.2(2026-03-10) {#cl-1.8.2}

- Fixed an issue where the `/logging/configs` endpoint returned an error in some scenarios (#85)
- Improved selected log messages to simplify troubleshooting the Operator's runtime status

## 1.8.1(2026-02-11) {#cl-1.8.1}

- Added support for injecting environment variables in `resourceFieldRef` format. Container resource limits and requests can now be referenced, including limits.cpu, limits.memory, requests.cpu, and requests.memory (#84)
- Added support for injecting the Python ddtrace agent (#82)
- Added a proxy API for retrieving Pod data from the local cluster (#79)

## 1.8.0(2026-01-29) {#cl-1.8.0}

- `admission_inject_v2` now includes the `check_annotation` field for compatibility with `admission.datakit/java-lib.version` usage (#81)
- Added compatibility with Profiler injection through the legacy `admission_inject.profiler` configuration (#81)

## 1.7.3(2026-01-08) {#cl-1.7.3}

- Added the `from_beginning_threshold_size` configuration field for the ClusterLoggingConfig CRD (#80)

## 1.7.2(2025-12-18) {#cl-1.7.2}

- Fixed a potential mount error when injecting logfwd (#78)

## 1.7.1(2025-12-17) {#cl-1.7.1}

- Fixed compatibility with legacy configurations (#77)
- In `admission_inject_v2`, injection is skipped when both `namespace_selectors` and `label_selectors` are empty (#77)
- Added the `enable_prometheus_annotations` option to Flameshot injection for adding Prometheus.io Annotations (#74)
- Changed the Flameshot port configuration from the `FLAMESHOT_HTTP_LOCAL_ADDR` environment variable to `FLAMESHOT_HTTP_LOCAL_PORT` (#75)
- Fixed the incorrect requirement to configure `log_configs` when injecting logfwd, because logfwd can use a CRD configuration source (#76)

## 1.7.0(2025-12-12) {#cl-1.7.0}

- Added `admission_inject_v2` to replace `admission_inject` while retaining backward compatibility (#73)
- Added Flameshot injection to replace the legacy Profiler injection (#72)
- When both `namespace_selectors` and `label_selectors` are configured, their relationship is now AND instead of OR
- Removed support for the `admission.datakit/logfwd.log_configs` and `admission.datakit/logfwd.volume_paths` Annotations (supported only in v1.6.X)
- Removed support for the `admission.datakit/java-lib.version` Annotation (injection can still be disabled with `admission.datakit/ddtrace.enabled:"false"`)

## 1.6.1(2025-12-02) {#cl-1.6.1}

- Improved error checking for the Kubernetes ClusterLoggingConfig CRD. When the resource does not exist, the Operator logs the condition and disables the feature (#71)

## 1.6.0(2025-11-19) {#cl-1.6.0}

- Added support for the Kubernetes ClusterLoggingConfig CRD and a new HTTP endpoint that returns log collection configurations (#70)
- Adjusted and improved logfwd injection for the new CRD-based log collection configuration (#70)

## 1.5.18(2025-07-15) {#cl-1.5.18}

- Added support for manually configuring Resources Requests and Resources Limits when injecting ddtrace (#68)

## 1.5.17(2025-06-03) {#cl-1.5.17}

- Added a UOS arm64 image release

## 1.5.16(2025-04-16) {#cl-1.5.16}

- Adjusted the matching order for injected logging configurations (#64)

## 1.5.15(2025-04-15) {#cl-1.5.15}

- Fixed Namespace matching support when deploying on Kubernetes 1.19 (#63)

## 1.5.14(2025-04-01) {#cl-1.5.14}

- Simplified Namespace regular expressions: use `*` instead of `.*` to match all Namespaces (#59)

## 1.5.13(2025-03-31) {#cl-1.5.13}

- Added regular expression matching for Namespaces and Labels when injecting DDtrace and Profiler (#58)
- Added regular expression matching for Namespaces and Labels when injecting logging configurations (#59)

## 1.5.12(2025-02-28) {#cl-1.5.12}

- Added support for manually configuring Resources Requests and Resources Limits when injecting logfwd and Profiler (#57)

## 1.5.11(2025-02-20) {#cl-1.5.11}

- Added support for adding the datakit/logs Annotation configuration to target Pods and automatically mounting directories (#53)
- Improved directory mount handling for logfwd Sidecar injection; existing mounts are now reused by default (#55)

## 1.5.10(2024-12-10) {#cl-1.5.10}

- Adjusted the injection order of the DD_TAGS environment variable so it can always reference preceding environment variable values (#51)

## 1.5.9(2024-11-25) {#cl-1.5.9}

- Changed the injected image pullPolicy to Always (#50)

## 1.5.8(2024-10-10) {#cl-1.5.8}

- Changed the ddtrace Python injection logic to inject only basic environment variables, without an image or library (#35)

## 1.5.7(2024-09-18) {#cl-1.5.7}

- Fixed incorrect DD_TAGS environment variable injection in v1.5.5 and v1.5.6 (#48)
- Added Annotations to Pods (`admission.datakit/ddtrace.enabled="false"`, `admission.datakit/logfwd.enabled="false"`, and `admission.datakit/profiler.enabled="false"`) for selectively disabling individual injection types (#49)

## 1.5.6(2024-09-13) {#cl-1.5.6}

- Improved Profiler volumeMount injection so a mount is not added when the same path already exists (#46)

## 1.5.5(2024-09-02) {#cl-1.5.5}

- Improved `DD_TAGS` environment variable injection for ddtrace. If `DD_TAGS` already exists in the original Pod, the new value is now appended instead of ignored (#45)

## 1.5.4(2024-08-27) {#cl-1.5.4}

- Removed resources from ddtrace injection and reduced resources for logfwd and Profiler (#44)

## 1.5.3(2024-04-09) {#cl-1.5.3}

- Fixed missing files in ddtrace mounts for multi-container Pods (#42)

## 1.5.2(2024-04-08) {#cl-1.5.2}

- Added batch ddtrace injection based on labelSelector (#41)

## 1.5.1(2024-04-07) {#cl-1.5.1}

- Added the current version and build information to logs (#41)

## 1.5.0(2024-03-18) {#cl-1.5.0}

- Added the `admission.datakit/enabled="false"` Annotation for disabling all injections on a Pod, including ddtrace, logfwd, and Profiler (#39)
- Added batch ddtrace injection for specified Namespaces (#38)
- Added the option to reuse volumes during logfwd injection, avoiding errors caused by mounting the same path multiple times (#34)

## 1.4.3(2023-12-21) {#cl-1.4.3}

- Fixed mount errors caused by wildcard paths in logfiles during logfwd injection (#31)
- Fixed injection failures that affected startup of the original Pod when it contained two or more containers (#32)
- Improved one log message

## 1.4.2(2023-09-18) {#cl-1.4.2}

- Fixed inconsistent ordering of injected environment variables (#28)

## 1.4.1(2023-09-15) {#cl-1.4.1}

- Added support for injecting environment variables into logfwd (#27)

## 1.4.0(2023-09-13) {#cl-1.4.0}

- Added environment variable configuration through Kubernetes Downward API FieldRef (#26)
- Added DD_TAGS by default for ddtrace (#26)

## 1.3.1(2023-08-24) {#cl-1.3.1}

- Updated the default Profiler image

## 1.3.0(2023-07-17) {#cl-1.3.0}

- Changed the minimum unit of Admission injection to a Pod; update the YAML to the latest version or datakit-operator-v1.3.0.yaml (#22)
- Added Profiler injection (#5)
- Added Resource Limits to injected Sidecars (#20)

## 1.2.1(2023-06-28) {#cl-1.2.1}

- Changed the default image registry (#19)
- Updated the Helm structure

## 1.2.0(2023-06-13) {#cl-1.2.0}

- Added JSON configuration for DataKit Operator while retaining compatibility with the existing environment variable configuration (#19)

## 1.0.5(2023-05-11) {#cl-1.0.5}

- Added a new ping API (#18)
- Added an API dedicated to DataKit election, implementing task distribution for DataKit collectors (#15)

## 1.0.4(2023-04-10) {#cl-1.0.4}

- Added logfwd injection through Kubernetes Admission (#12)
- Added the `DD_AGENT_HOST` and `DD_TRACE_AGENT_PORT` environment variables by default when injecting the ddtrace agent (#14)
- Changed the list of resources supported by Admission; native Pods are no longer supported (#12)
- Restructured the code and completed unit tests for ddtrace and logfwd
- Improved the datakit.yaml structure
- Removed the docs directory and its documentation, and added new documentation links to the README

## 1.0.3(2023-03-27) {#cl-1.0.3}

- Added English documentation
- Added dd-agent image address configuration through environment variables (#9)
- Improved several environment variable names
- Changed datakit.yaml to ignore webhook errors by default (#11)

## 1.0.2(2023-03-09) {#cl-1.0.2}

- Added the CHANGELOG (#9)
- Fixed access failures caused by expired certificates by regenerating self-signed certificates with a longer validity period (#8)
- Fixed a minor error encountered when publishing images (#10)
- Changed YAML installation so it no longer creates the Namespace, and documented the required step (#7)

## 1.0.1(2022-12-28) {#cl-1.0.1}

- Added Makefile, Dockerfile, and CI configuration (#2)
- Added ddtrace file and environment variable injection through Kubernetes Admission (#1)
