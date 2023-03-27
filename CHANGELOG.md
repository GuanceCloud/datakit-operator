# Changelog

## [1.0.3] - 2023-03-27

### Added

- 支持以环境变量的方式配置 dd-agent 镜像地址 (#9)
- 添加英文文档

### Changed

- 优化几个环境变量的命名
- 修改 datakit.yaml，默认忽略 webhook 的报错 (#11)

## [1.0.2] - 2023-03-09

### Added

- 添加 CHANGELOG (#9)

### Changed

- 修改 yaml 安装方式，不在 yaml 中创建 namespace，在文档中补全说明 (#7)

### Fixed

- 修复因证书过期导致访问失败的问题，重新生成自签证书且延长过期时间 (#8)
- 修复发布 image 遇到的一个细节错误 (#10)

## [1.0.1] - 2022-12-28

### Added

- 添加 Makefile、Dockerfile 和 CI 配置 (#2)
- 支持以 k8s admission 方式注入 dd-lib 文件和 env (#1)
