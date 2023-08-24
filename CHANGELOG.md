# Changelog

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
