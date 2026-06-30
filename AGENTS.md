# AGENTS.md

本文件给后续接手 Hermes Dock 的 Agent 使用。回答用户时使用中文，代码、命令、文件名和配置键保持原文。

## 项目定位

Hermes Dock 是一个 Wails 桌面启动器，用来管理当前用户下单个 Hermes Agent Docker 实例。实例目录固定为：

```text
~/.hermes-dock
```

目标用户主要是中国大陆的新手用户。优先降低命令行暴露，界面文案使用简体中文。

## 核心边界

- 只管理本机单实例，不做多实例。
- 多 profile 目标仍是单 Docker 容器，不做多 Compose project 或多容器实例。
- 只要求用户已安装 Docker，不负责安装 Docker。
- 启动器接管标准 `~/.hermes-dock/docker-compose.yaml`。
- 高级 Docker 自定义放在 `~/.hermes-dock/docker-compose.override.yaml`。
- `~/.hermes-dock/data` 是用户数据，默认永不覆盖。
- 只做显式保存、绑定或迁移，不做静默重置。
- 不把真实运行态、日志、会话、缓存、数据库、auth 文件或微信账号凭据放进内置模板。
- 多 profile 第一版不为每个 profile 单独暴露 HTTP/API/Dashboard 端口；profile gateway 只服务各自平台入口。

## 重要目录

```text
templates/seed-data/       内置干净模板，首次启动释放到 data/
frontend/src/App.tsx       React 主界面
frontend/src/App.css       React 样式
app.go                     Wails 状态聚合
compose.go                 Compose 生成和容器生命周期
config.go                  config.yaml、模型配置和模型列表
env.go                     .env 读写、合并和脱敏
weixin.go                  个人微信扫码登录
platforms.go               企业微信、飞书和通道相关操作
paths.go                   实例路径和 safePath 限制
backup.go                  写入前备份
```

多 profile 相关路径约定：

```text
~/.hermes-dock/data/                     default profile 的 Hermes home
~/.hermes-dock/data/profiles/<id>/       非 default profile 的 Hermes home
~/.hermes-dock/launcher/profiles.json    Dock profile registry，事实来源
~/.hermes-dock/data/.dock/               runner 派生运行态
```

项目不保留单独的 `docs/` 目录。架构和 MVP 边界必须维护在 `README.md` 和本文件中，避免多份文档漂移。

## 数据安全规则

- 修改任何会影响 `~/.hermes-dock/data` 的逻辑前，先确认不会覆盖用户已有文件。
- `releaseSeedData` 只能创建缺失文件，不能覆盖已有文件。
- 写入 `config.yaml`、`.env`、`docker-compose.yaml` 或高级编辑文件前，应保留备份。
- 写入 profile 的 `config.yaml`、`.env`、`SOUL.md`、`skills/` 或 `launcher/profiles.json` 前，应保留备份。
- 删除非 default profile 前，必须先整体打包备份 profile 目录；备份失败则中止删除。
- 不要把密钥写入 `launcher/state.json`。
- 不要把密钥写入 `launcher/profiles.json`、`data/.dock/profiles-runtime.json` 或 `data/.dock/profile-status.json`。
- UI 日志、事件、错误信息中不要输出完整 token、API key、secret。
- 不要为了兼容失败而吞掉错误；应返回清晰错误，让 UI 展示。
- 高级页“恢复出厂设置”是唯一允许删除整个 `~/.hermes-dock` 的流程；必须先 `docker compose down`，失败则中止，不加 `--volumes`。

## Compose 约定

`docker-compose.yaml` 由启动器生成和维护，当前模板包含：

- Hermes 镜像。
- 单 profile 版本为 `command: gateway run`；多 profile 版本应启动 Hermes Dock runner，由 runner 并行启动多个 profile gateway worker。
- 控制台和网关端口。
- 控制台认证环境变量，控制台固定启用。
- 中国大陆友好的 pip、uv、npm 镜像源。
- 单 profile 版本使用 `env_file: ./data/.env`；多 profile runner 版本不能用一个全局 `env_file` 表达 profile 运行态密钥。
- `volumes: ./data:/opt/data`。

容器操作命令约定：

- 启动：`docker compose up -d`
- 停止：`docker compose stop`
- 重启：`docker compose restart`
- 重建：`docker compose up -d --force-recreate`

不要用普通 `docker compose restart` 作为“应用配置”的实现，因为 Docker 不会刷新已创建容器的环境变量；配置变更需要通过“重建”应用。

多 profile 版本中，`SOUL.md`、`skills/`、`config.yaml`、`.env` 和平台绑定保存后也不承诺热更新，统一通过“应用并重建”进入运行态。

## 架构约定

- Go 后端执行 Docker、文件、备份、平台绑定和模型列表拉取。
- React 前端只保留表单状态和展示状态，保存动作走 Wails Go 方法。
- Wails 事件用于流式输出 Docker 日志、命令进度和微信扫码状态。
- 内置模板来自 `templates/seed-data/`，只能包含干净初始文件和 Hermes 内置 skills 快照。
- `launcher/state.json` 只保存启动器元数据和 UI 策略，不保存密钥。
- `launcher/profiles.json` 保存 profile registry，不保存密钥，数组顺序就是 UI 显示顺序。
- `data/.dock/profiles-runtime.json` 是可再生成 runner manifest，不备份。
- `data/.dock/profile-status.json` 是 runner 写入的运行态状态，不备份。
- `launcher/backups/` 保存写入前备份。
- `launcher/helpers/` 保存临时 helper，例如微信扫码登录脚本。

## 多 Profile 设计约定

目标模型：

- 一个 Docker Compose service。
- 一个 Hermes 容器。
- 一个 Hermes Dock runner 作为容器主进程。
- 多个 Hermes profile gateway 子进程并行运行。
- 每个子进程绑定一个 Hermes profile，服务该 profile 的个人微信、企业微信或飞书 / Lark 入口。

profile 目录和 ID：

- `default` profile 使用 `data/` 根目录，默认注册并启用，但允许停用，不允许删除。
- 非 default profile 使用 `data/profiles/<id>/`。
- profile ID 只能使用路径安全 ASCII slug：`a-z`、`0-9`、`-`，创建后不可改；中文只作为显示名。
- 删除非 default profile 后允许重建同名 ID，但如果残留目录存在，不自动复用或覆盖。

profile 隔离：

- 按 profile 隔离：`SOUL.md`、`skills/`、`config.yaml`、`.env`、供应商、模型、平台绑定、通道目录、记忆和会话。
- 全局共享：Docker 镜像、容器名、端口、CPU、内存、shm、`docker-compose.override.yaml`。
- 模型供应商和 API Key 默认 profile 级隔离。UI 可以提供显式复制模型配置到其他 profile，默认不复制 API Key。
- 平台策略如 `WECOM_DM_POLICY`、`WECOM_GROUP_POLICY`、`WEIXIN_DM_POLICY`、`FEISHU_GROUP_POLICY` 也按 profile 写入各自 `.env`。

profile 创建：

- 默认创建干净 profile，只释放模板和必要账号目录，不复制密钥、平台账号、记忆、会话或通道目录。
- 第一版可支持“从当前 profile 复制人格和 skills”，不做完整克隆。
- 非 default profile 创建时，`config.yaml` 的 `terminal.cwd` 应指向 `/opt/data/profiles/<id>`。
- 非 default profile 创建时，`SOUL.md` 中面向模型的工作目录说明应指向 `/opt/data/profiles/<id>` 和 `/opt/data/profiles/<id>/tmp`。
- 不批量改写 `skills/` 内部文档路径。
- 每个 profile `.env` 可写入非敏感标识，例如 `HERMES_DOCK_PROFILE` 和 `HERMES_DOCK_PROFILE_HOME`。

runner 行为：

- runner 统一设置 `HERMES_HOME=/opt/data`。
- default 启动为 `hermes gateway run`，非 default 启动为 `hermes -p <id> gateway run`，优先使用 Hermes 原生 profile 解析。
- runner 必须安全解析每个 profile 的 `.env`，禁止用 shell `source` 执行用户可编辑文件。
- enabled 且有完整平台绑定的 profile 才启动。
- enabled 但未绑定平台的 profile 不启动，状态为未绑定平台，不视为严重错误。
- 平台绑定不完整、enabled profile 平台身份冲突、`config.yaml` 无法解析等严重错误应在“应用并重建”前阻止。
- 同一个 `WECOM_BOT_ID`、`WEIXIN_ACCOUNT_ID` 或 `FEISHU_APP_ID` 不能被多个 enabled profile 同时使用。
- 一个 profile 可以同时绑定个人微信、企业微信和飞书 / Lark。
- runner 给每行子进程日志加 profile 前缀，例如 `[sales] ...`；runner 自身日志使用 `[runner]`。
- runner 对异常退出的 profile 做有限自动重启，连续失败后标记 failed，其他 profile 继续运行。
- 无可运行 profile 时 runner 仍保持容器 running。
- 第一版停止、启动、重启仍作用于整个容器，不做单 profile 启停/重启。

UI 和功能边界：

- 首页应以 Profiles 总览为主，同时展示容器运行环境状态。
- profile 优先，平台绑定在 profile 详情内完成。
- `enabled` 只表示参与运行，不影响编辑。
- 保存配置不自动重建；显示未应用变更，由用户手动“应用并重建”。
- 模型测试和平台测试消息只针对当前 profile。
- 高级编辑以当前 profile 为上下文，打开当前 profile 的 `config.yaml`、`.env`、`SOUL.md` 或 `skills/`。
- 第一版不做 Kanban/跨 profile 协作 UI，但目录、ID 和 runner 启动方式必须保持 Hermes 原生 profile/Kanban 兼容。
- 第一版不做按消息内容跨 profile 路由，不做 profile 导入/导出，不做批量创建向导，不做 skills marketplace。

测试要求：

- 多 profile 实现应补 Go 单元测试，至少覆盖 profile ID 校验、registry 读写、路径解析、创建 profile、平台身份唯一性、runtime manifest 生成和 runner env/log/restart 关键逻辑。
- 实现完成后允许运行 `go test ./...`；如改 Wails 绑定或前端类型，也运行 `wails generate module` 和 `pnpm --dir frontend run build`。

## MVP 范围

当前包含：

- Docker / Compose 检测。
- 首次启动初始化。
- 标准 compose 生成和 override 入口。
- 启动、停止、重启、重建、状态和日志。
- 部署配置、主模型和 auxiliary 模型配置，平台配置通过结构化页面写入 `.env`。
- DashScope 按量计费和 DeepSeek 供应商预设及模型列表拉取。
- 个人微信扫码登录。
- 企业微信 AI Bot WebSocket 配置。
- 飞书 / Lark WebSocket 配置。
- 通道目录查看、默认通道设置和测试消息。
- UI 输出脱敏和本地备份。

当前不做：

- Docker 安装。
- 系统服务安装。
- 多实例管理。
- 当前稳定版本不做多账号平台管理；下一阶段通过单容器多 profile 支持多个平台身份并行。
- 内置聊天客户端。
- 在线 Docker tag 浏览。
- 云同步或远程备份。
- 任意 Hermes 平台完整配置器。
- 按通道配置 prompt、模型或工具路由。

## 模型配置

供应商配置是独立模块，事实来源是当前 profile `config.yaml` 顶层 `providers`。单 profile 版本路径为 `data/config.yaml`。`model.provider` 和 `auxiliary.<name>.provider` 只保存供应商 ID；保存时启动器会从 `providers` 展开 `base_url`、`api_mode` 和 `api_key` 到 `model` / `auxiliary`，兼容 Hermes 当前运行态。不要把模型页重新做成 API Key、Base URL 和模型名混在一起的表单。

内置供应商实例：

- `dashscope-payg`：DashScope 按量计费，`provider: custom`，默认模型 `qwen3.7-max`，模型列表 `https://dashscope.aliyuncs.com/compatible-mode/v1/models`。
- `opencode-go`：OpenCode Go，`provider: custom`，默认模型 `deepseek-v4-flash`，模型列表 `https://opencode.ai/zen/go/v1/models`。
- `deepseek`：DeepSeek，`provider: deepseek`，默认模型 `deepseek-v4-flash`，模型列表 `https://api.deepseek.com/models`。

供应商页负责新增、编辑、禁用供应商和填写 API Key。内置供应商不可删除，自定义供应商被主模型或辅助模型引用时不可删除。模型页只选择供应商 ID 和模型名；主供应商没有 API Key 时允许保存模型选择，但禁止测试。保存供应商或模型配置时，只把当前主模型和辅助模型实际引用的供应商密钥同步到当前 profile `.env`，空密钥不同步，也不清理旧 `.env` 遗留键。

Auxiliary 模型策略由 UI 控制，状态记录在 `launcher/state.json` 的 `ModelAuxiliaryMode`。`auto` 策略下辅助模型保持 `provider: auto` 和空兼容字段；`follow-main` 和 `custom` 才根据引用供应商展开兼容字段。

## 平台绑定

当前稳定版本每个平台只支持一个实例；多 profile 第一版会改为每个 profile 可绑定一个实例，enabled profiles 中平台身份必须唯一：

- 一个 Weixin / WeChat Personal。
- 一个 WeCom AI Bot。
- 一个 Feishu / Lark App。

个人微信：

- 使用短生命周期 Docker helper 运行扫码登录。
- helper 输出 NDJSON，Go 层解析事件。
- token 不返回给 UI。
- 扫码成功后写入当前 profile `.env` 和 `weixin/accounts/`。
- 多 profile 版本中扫码成功后不自动重建，用户手动应用；当前稳定版本仍可自动应用配置并重建 gateway 容器。
- 默认 `WEIXIN_DM_POLICY=open`。
- 默认 `WEIXIN_GROUP_POLICY=open`。

企业微信：

- 只支持企业微信 AI Bot WebSocket。
- 默认 `WECOM_DM_POLICY=open`。
- 默认 `WECOM_GROUP_POLICY=open`。
- 多 profile 版本中 `WECOM_BOT_ID` 在 enabled profiles 中必须唯一。

飞书 / Lark：

- 只支持 WebSocket 模式，手动填写 App ID 和 App Secret，不做 webhook。
- 默认 `FEISHU_DOMAIN=feishu`，可切换 `lark`。
- 固定写入 `FEISHU_CONNECTION_MODE=websocket`。
- 默认 `FEISHU_GROUP_POLICY=allowlist`。
- 多 profile 版本中 `FEISHU_APP_ID` 在 enabled profiles 中必须唯一。

## 前端约定

- 界面文案使用简体中文。
- 面向操作工具，不做营销型 landing page。
- 设计应克制、清晰、密度适中。
- 不要使用不必要的分隔线。
- 操作按钮优先使用 lucide-react 图标。
- 不要把说明性大段文字塞进主界面；说明放 `README.md` 或 `AGENTS.md`。
- 保证移动和桌面窗口下按钮文字不溢出、不重叠。
- 不在普通导航中暴露环境变量编辑页；需要写入 `.env` 时优先走模型、部署或平台绑定等结构化表单，高级编辑可打开 `data/.env`。
- 多 profile 版本中，高级编辑必须清楚显示当前 profile，避免编辑错 profile 文件。

## 开发命令

```bash
pnpm --dir frontend install
wails generate module
wails dev
go test ./...
pnpm --dir frontend run build
```

用户明确要求验证时再运行测试或构建。文档类修改通常不需要主动跑测试。

## 代码风格

- Go 代码改动后运行 `gofmt`。
- 前端依赖使用 `pnpm`。
- Python 临时代码如必须使用，遵循用户级约定：用 `uv run python`，不要直接用 `python` 或 `pip`。
- 优先保持改动小而直接，避免过度抽象。
- 不做与任务无关的格式化。
- 不使用破坏性 git 命令。

## 常见坑

- `.env` 变化后，已创建容器不会自动刷新环境变量，必须重建容器。
- Hermes CLI 可能能从 `/opt/data/.env` 读到配置，但 gateway 运行态依赖进程环境变量。
- `docker compose config` 能看到 env 并不代表当前旧容器已经拿到 env。
- 多 profile 版本不能继续依赖全局 Compose `env_file`，否则多个 profile 的同名平台密钥会互相覆盖。
- 非 default profile 如果 `terminal.cwd` 或 `SOUL.md` 仍指向 `/opt/data`，会污染 default profile 根目录。
- Weixin iLink bot 是否能收到普通微信群消息，受 iLink 侧能力限制。
- 现有 `~/.hermes-dock/data/.env` 可能包含旧版本遗留键，默认保留，不要清理用户文件。
