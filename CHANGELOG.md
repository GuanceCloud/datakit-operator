# Changelog

## [1.5.9] - 2024-11-25

- 修改注入镜像的 pullPolicy 为 Always（#50）

## [1.5.8] - 2024-10-10

- 修改 ddtrace Python 的注入逻辑，不再注入镜像和 lib，只添加基础环境变量（#35）

## [1.5.7] - 2024-09-18

- 修复 v1.5.5 和 v1.5.6 注入环境变量 DD_TAGS 的错误问题（#48）
- 支持在 Pod 上添加 Annotations（`admission.datakit/ddtrace.enabled="false"`、`admission.datakit/logfwd.enabled="false"` 和 `admission.datakit/profiler.enabled="false"`）更细致地关闭某一类注入（#49）

## [1.5.6] - 2024-09-13

- 优化注入 profiler volumeMount 逻辑，如果已经存在相同的 path 就不再注入（#46）

## [1.5.5] - 2024-09-02

- 优化注入 ddtrace 环境变量 `DD_TAGS` 逻辑，如果原 Pod 已经存在 `DD_TAGS`，现在会追加而不是忽略（#45）

## [1.5.4] - 2024-08-27

- 移除在注入 ddtrace 的 resource，降低 logfwd 和 profiler 的 resource（#44）

## [1.5.3] - 2024-04-09

- 修复在多容器情况下 ddtrace 挂载文件缺失的问题（#42）

## [1.5.2] - 2024-04-08

- 支持根据 labelSelector 批量注入 ddtrace（#41）

## [1.5.1] - 2024-04-07

- 在日志打印当前版本和编译信息（#41）

## [1.5.0] - 2024-03-18

- 支持在 Pod 上添加 Annotation `admission.datakit/enabled="false"` 关闭所有的注入，包括注入 ddtrace、logfwd 和 profiler（#39）
- 支持给指定的 namespace 批量注入 ddtrace（#38）
- 在注入 logfwd 时可以选择复用 volume，避免同路径多次挂载的报错问题（#34）

## [1.4.3] - 2023-12-21

- 修复在注入 logfwd 时，如果 logfiles 写通配路径会导致 mount 错误的问题（#31）
- 修复在注入 logfwd 时，如果该 Pod 有 2 个及以上的容器，会注入失败并影响原 Pod 启动的问题（#32）
- 优化一处日志打印

## [1.4.2] - 2023-09-18

- 修复注入的环境变量顺序不一致问题 (#28)

## [1.4.1] - 2023-09-15

- 支持给 logfwd 注入环境变量 (#27)

## [1.4.0] - 2023-09-13

- 支持以 Kubernetes DownloadAPI FieldRef 的方式配置环境变量 (#26)
- ddtrace 默认添加 DD_TAGS (#26)

## [1.3.1] - 2023-08-24

- 更新默认 Profiler image

## [1.3.0] - 2023-07-17

- 修改 admission 注入的最小单元为 Pod，需要将 yaml 更新到最新或 datakit-operator-v1.3.0.yaml (#22)
- 支持注入 profiler (#5)
- 注入的 sidecar 添加 Resource Limit (#20)

## [1.2.1] - 2023-06-28

- 修改默认的镜像仓库为 guance.com (#19)
- 更新 Helm 结构

## [1.2.0] - 2023-06-13

- 支持以 json-config 的方式配置 Datakit Operator，现有的环境变量方式保持兼容 (#19)

## [1.0.5] - 2023-05-11

- 添加新的 ping API (#18)
- 添加 Datakit 选举专用 API，实现对 Datakit 采集器的任务分发 (#15)

## [1.0.4] - 2023-04-10

- 支持以 Kubernetes Admission 方式注入 logfwd 程序 (#12)
- 注入 ddtrace agent 时会默认添加 `DD_AGENT_HOST` 和 `DD_TRACE_AGENT_PORT` 两个环境变量 (#14)
- 修改 Admission 支持的 resources 列表，不再支持原生 Pod (#12)
- 修改代码的结构，补全 ddtrace 和 logfwd 的单元测试
- 优化 datakit.yaml 的结构
- 移除 docs 目录和文档，在 README 提供新的文档链接

## [1.0.3] - 2023-03-27

- 添加英文文档
- 支持以环境变量的方式配置 dd-agent 镜像地址 (#9)
- 优化几个环境变量的命名
- 修改 datakit.yaml，默认忽略 webhook 的报错 (#11)

## [1.0.2] - 2023-03-09

- 添加 CHANGELOG (#9)
- 修复因证书过期导致访问失败的问题，重新生成自签证书且延长过期时间 (#8)
- 修复发布 image 遇到的一个细节错误 (#10)
- 修改 yaml 安装方式，不在 yaml 中创建 namespace，在文档中补全说明 (#7)

## [1.0.1] - 2022-12-28

- 添加 Makefile、Dockerfile 和 CI 配置 (#2)
- 支持以 Kubernetes Admission 方式注入 ddtrace 文件和环境变量 (#1)
