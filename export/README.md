# DataKit Operator 文档导出

`export/zh` 是 DataKit Operator 基础文档的基准目录。`export/en`、`export/ja` 和 `export/ko` 由中文文档生成并提交到仓库，四个目录应保持相同的文件名、模板变量、品牌变量和固定锚点。标准组件镜像及少量非镜像版本统一配置在 `config.env`。

Operator 始终导出四种语言；下游站点按品牌发布：Guance 发布中文、英文、日文和韩文，TrueWatch 发布英文、日文和韩文。

修改中文文档后，使用仓库中的 Codex Skill 同步英文、日文和韩文：

```text
$translate-operator-docs
```

翻译完成后会更新各语言目录中的 `.translation/metadata.json`。译文和 `metadata.json` 需要一起提交。

在仓库根目录运行以下命令，将文档导出到 `dataflux-doc` 的既有位置：

```shell
./export.sh
./export.sh -D /path/to/dataflux-doc
```

默认目标是 `~/git/dataflux-doc`。导出器只覆盖 `docs/{zh,en,ja,ko}/datakit` 下与本目录同名的文档，不删除其他文件、不修改导航，也不执行 Git 操作。`<<<...>>>` 品牌变量会保持原样，由 `dataflux-doc` 的 MkDocs 在发布时渲染。

提交前运行 `make docs_lint` 检查源文档及渲染结果。该命令通过 `./export.sh -c` 在临时目录完成渲染，不要求本地存在 `dataflux-doc`，也不会修改文档源文件。

如果新增或删除文档并需要调整导航，还需要同步修改 DataKit 仓库中对应语言的 `datakit.pages`。
