# DataKit Operator 文档导出

`export/zh` 和 `export/en` 是 DataKit Operator 基础文档的源文件，两个目录应保持相同的文件名和模板变量。标准组件镜像及少量非镜像版本统一配置在 `config.env`。

在仓库根目录运行以下命令，将文档导出到 `dataflux-doc` 的既有位置：

```shell
./export.sh
./export.sh -D /path/to/dataflux-doc
```

默认目标是 `~/git/dataflux-doc`。导出器只覆盖 `docs/{zh,en}/datakit` 下与本目录同名的文档，不删除其他文件、不修改导航，也不执行 Git 操作。`<<<...>>>` 品牌变量会保持原样，由 `dataflux-doc` 的 MkDocs 在发布时渲染。

提交前运行 `make docs_lint` 检查源文档及渲染结果。该命令通过 `./export.sh -c` 在临时目录完成渲染，不要求本地存在 `dataflux-doc`，也不会修改文档源文件。

如果新增或删除文档并需要调整导航，还需要同步修改 DataKit 仓库中的 `internal/export/doc/zh/datakit.pages` 和 `internal/export/doc/en/datakit.pages`。
