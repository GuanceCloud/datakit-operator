# Changelog

## [1.0.2] - 2023-03-09

### Added

- 添加 CHANGELOG (#9)

### Changed

- 修改 yaml 安装方式，不在 yaml 中创建 namespace，在文档中补全说明 (#7)

### Fixed

- 修复因证书过期导致访问失败的问题，重新生成自签证书且延长过期时间 (#8)

## [1.0.1] - 2022-12-28

### Added

- 添加 Makefile、Dockerfile 和 CI 配置 (#2)
- 支持以 k8s admission 方式注入 dd-lib 文件和 env (#1)
