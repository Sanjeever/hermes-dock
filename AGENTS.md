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
- 只要求用户已安装 Docker，不负责安装 Docker。
- 启动器接管标准 `~/.hermes-dock/docker-compose.yaml`。
- 高级 Docker 自定义放在 `~/.hermes-dock/docker-compose.override.yaml`。
- `~/.hermes-dock/data` 是用户数据，默认永不覆盖。
- 只做显式保存、绑定或迁移，不做静默重置。
- 不把真实运行态、日志、会话、缓存、数据库、auth 文件或微信账号凭据放进内置模板。

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
platforms.go               企业微信和通道相关操作
paths.go                   实例路径和 safePath 限制
backup.go                  写入前备份
```

项目不保留单独的 `docs/` 目录。架构和 MVP 边界必须维护在 `README.md` 和本文件中，避免多份文档漂移。

## 数据安全规则

- 修改任何会影响 `~/.hermes-dock/data` 的逻辑前，先确认不会覆盖用户已有文件。
- `releaseSeedData` 只能创建缺失文件，不能覆盖已有文件。
- 写入 `config.yaml`、`.env`、`docker-compose.yaml` 或高级编辑文件前，应保留备份。
- 不要把密钥写入 `launcher/state.json`。
- UI 日志、事件、错误信息中不要输出完整 token、API key、secret。
- 不要为了兼容失败而吞掉错误；应返回清晰错误，让 UI 展示。

## Compose 约定

`docker-compose.yaml` 由启动器生成和维护，当前模板包含：

- Hermes 镜像。
- `command: gateway run`。
- 控制台和网关端口。
- 控制台认证环境变量，控制台固定启用。
- 中国大陆友好的 pip、uv、npm 镜像源。
- `env_file: ./data/.env`。
- `volumes: ./data:/opt/data`。

容器操作命令约定：

- 启动：`docker compose up -d`
- 停止：`docker compose stop`
- 重启：`docker compose restart`
- 重建：`docker compose up -d --force-recreate`

不要用普通 `docker compose restart` 作为“应用配置”的实现，因为 Docker 不会刷新已创建容器的环境变量；配置变更需要通过“重建”应用。

## 架构约定

- Go 后端执行 Docker、文件、备份、平台绑定和模型列表拉取。
- React 前端只保留表单状态和展示状态，保存动作走 Wails Go 方法。
- Wails 事件用于流式输出 Docker 日志、命令进度和微信扫码状态。
- 内置模板来自 `templates/seed-data/`，只能包含干净初始文件和 Hermes 内置 skills 快照。
- `launcher/state.json` 只保存启动器元数据和 UI 策略，不保存密钥。
- `launcher/backups/` 保存写入前备份。
- `launcher/helpers/` 保存临时 helper，例如微信扫码登录脚本。

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
- 通道目录查看、默认通道设置和测试消息。
- UI 输出脱敏和本地备份。

当前不做：

- Docker 安装。
- 系统服务安装。
- 多实例管理。
- 多账号平台管理。
- 内置聊天客户端。
- 在线 Docker tag 浏览。
- 云同步或远程备份。
- 任意 Hermes 平台完整配置器。
- 按通道配置 prompt、模型或工具路由。

## 模型配置

模型 API Key 存在 `data/config.yaml` 的 `model.api_key` 中，不提供环境变量页，也不要求用户填写供应商环境变量。保存模型配置时，启动器会自动把供应商密钥同步到 `data/.env` 的 `DASHSCOPE_API_KEY`、`OPENCODE_GO_API_KEY` 或 `DEEPSEEK_API_KEY`，供容器运行态读取。

内置供应商：

- DashScope 按量计费：
  - `provider: custom`
  - `base_url: https://dashscope.aliyuncs.com/compatible-mode/v1`
  - `api_mode: chat_completions`
  - 默认模型 `qwen3.7-max`
  - 拉取模型列表使用 `https://dashscope.aliyuncs.com/compatible-mode/v1/models`
- OpenCode Go：
  - `provider: custom`
  - `base_url: https://opencode.ai/zen/go/v1`
  - `api_mode: chat_completions`
  - 默认模型 `deepseek-v4-flash`
  - 拉取模型列表使用 `https://opencode.ai/zen/go/v1/models`
- DeepSeek：
  - `provider: deepseek`
  - `base_url: https://api.deepseek.com`
  - `api_mode: chat_completions`
  - 默认模型 `deepseek-v4-flash`
  - 拉取模型列表使用 `https://api.deepseek.com/models`

Auxiliary 模型策略由 UI 控制，状态记录在 `launcher/state.json` 的 `ModelAuxiliaryMode`。

模型页支持自定义 OpenAI 兼容供应商。自定义供应商保存到 `config.yaml` 的 `model.provider: custom`、`model.base_url`、`model.api_mode`、`model.default` 和 `model.api_key`；拉取模型列表时从用户填写的接口地址推导 `/models`，失败时允许用户手动填写模型名。

## 平台绑定

MVP 每个平台只支持一个实例：

- 一个 Weixin / WeChat Personal。
- 一个 WeCom AI Bot。

个人微信：

- 使用短生命周期 Docker helper 运行扫码登录。
- helper 输出 NDJSON，Go 层解析事件。
- token 不返回给 UI。
- 扫码成功后写入 `.env` 和 `data/weixin/accounts/`。
- 扫码成功后应自动应用配置并重建 gateway 容器。
- 默认 `WEIXIN_DM_POLICY=open`。
- 默认 `WEIXIN_GROUP_POLICY=open`。

企业微信：

- 只支持企业微信 AI Bot WebSocket。
- 默认 `WECOM_DM_POLICY=open`。
- 默认 `WECOM_GROUP_POLICY=open`。

## 前端约定

- 界面文案使用简体中文。
- 面向操作工具，不做营销型 landing page。
- 设计应克制、清晰、密度适中。
- 不要使用不必要的分隔线。
- 操作按钮优先使用 lucide-react 图标。
- 不要把说明性大段文字塞进主界面；说明放 `README.md` 或 `AGENTS.md`。
- 保证移动和桌面窗口下按钮文字不溢出、不重叠。
- 不在普通导航中暴露环境变量编辑页；需要写入 `.env` 时优先走模型、部署或平台绑定等结构化表单，高级编辑可打开 `data/.env`。

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
- Weixin iLink bot 是否能收到普通微信群消息，受 iLink 侧能力限制。
- 现有 `~/.hermes-dock/data/.env` 可能包含旧版本遗留键，默认保留，不要清理用户文件。
