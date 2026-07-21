# Hermes Dock 构建资产

`build/` 保存 Wails 为 Hermes Dock 打包桌面应用时使用的平台资产；这些文件属于发布输入，不是运行时生成的数据。

- `appicon.png` 和 `appicon.svg`：应用图标源文件。
- `darwin/`：macOS 发布构建使用的 `Info.plist`，以及开发模式使用的 `Info.dev.plist`。
- `windows/`：Windows manifest、版本信息、图标和 NSIS 安装器脚本。
- `bin/`：本地构建输出目录（如存在），不作为源码提交。

修改平台资产时，应同时核对 `wails.json`、对应平台安装器配置和发布 workflow，避免应用元数据、版本号或安装行为不一致。正式发布约束和版本同步规则维护在根目录的 `README.md` 与 `AGENTS.md`。
