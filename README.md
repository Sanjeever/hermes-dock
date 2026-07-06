# Hermes Dock

Hermes Dock 是一个面向本地单实例 Hermes Agent 的桌面启动器。它基于 Wails 构建，用一个可视化界面管理 `~/.hermes-dock` 下的 Hermes Docker 实例，让不熟悉命令行的新手也能完成初始化、模型配置、平台绑定、启动、停止、重启和重建。

项目目标很明确：只要用户已经安装 Docker，就可以打开 Hermes Dock，完成必要配置，然后启动 Hermes Agent。

## 当前能力

- 首次启动自动创建 `~/.hermes-dock`。
- 释放干净模板到 `~/.hermes-dock/data`，不捆绑当前真实运行态。
- 捆绑 Hermes 内置 skills 快照。
- 生成并接管标准 `docker-compose.yaml`。
- 保留 `docker-compose.override.yaml` 作为高级覆盖入口。
- 可视化管理 Docker 镜像、端口、控制台账号密码、内存、CPU 和共享内存，控制台固定启用。
- 通过模型、部署和平台绑定表单写入必要配置，不向普通用户提供环境变量编辑页。
- 可视化配置主模型和 auxiliary 模型。
- 支持 DashScope 按量计费和 DeepSeek 两个模型供应商预设。
- 支持通过 API Key 拉取模型列表并选择模型。
- 支持个人微信 Weixin / WeChat Personal 扫码登录。
- 支持企业微信 AI Bot WebSocket 配置。
- 支持飞书 / Lark WebSocket 配置。
- 支持查看通道目录、设置默认通道、发送测试消息。
- 内置 Web 管理界面，随桌面主进程启动，默认局域网可访问。
- 写入托管文件前自动备份。
- UI 日志和事件会脱敏敏感字段。

## 架构概览

Hermes Dock 的运行模型是“桌面启动器 + 本地 Docker Compose 单实例”：

- Go 后端负责文件读写、备份、Docker Compose 命令、模型列表拉取和平台绑定 helper。
- React 前端负责表单、状态展示、扫码流程、日志输出和通道管理。
- Wails 事件用于推送 Docker 输出、日志行和微信扫码状态。
- 内置 Web 管理服务运行在桌面主进程内，通过 HTTP RPC 和 WebSocket 复用主要管理能力。
- Hermes 容器只通过 `./data:/opt/data` 访问用户数据。
- 启动器自己的状态只保存在 `launcher/` 下。

下一阶段的多 profile 设计仍保持单 Docker 容器，不做多实例。容器内由 Hermes Dock runner 并行启动多个 Hermes profile gateway worker，每个 profile gateway 只服务自己绑定的平台入口，不单独向宿主机暴露 HTTP/API/Dashboard 端口。

内置模板来自 `templates/seed-data/`，只包含干净初始文件：

- `config.yaml`
- `.env`
- `SOUL.md`
- Hermes 内置 skills 快照
- 必要的空目录

内置模板明确不包含：

- 日志
- 缓存
- 数据库
- 会话
- auth 文件
- 微信账号凭据
- 当前机器的真实运行态

## 数据目录

Hermes Dock 固定管理当前用户下的单实例目录：

```text
~/.hermes-dock/
  docker-compose.yaml
  docker-compose.override.yaml
  data/
    config.yaml
    .env
    SOUL.md
    skills/
    weixin/
    profiles/
      sales/
        config.yaml
        .env
        SOUL.md
        skills/
        weixin/
      support/
        config.yaml
        .env
        SOUL.md
        skills/
        weixin/
    .dock/
      profiles-runtime.json
      profile-status.json
  launcher/
    state.json
    profiles.json
    web-server.json
    web-sessions.json
    logs/
      web-server.log
    backups/
    helpers/
```

`data/` 是用户数据，也是 `default` profile 的 Hermes home。非 default profile 使用 `data/profiles/<id>/`，保持和 Hermes 原生 profile 结构兼容。Hermes Dock 默认不会覆盖已有用户数据，只在明确保存配置、绑定平台或执行迁移时写入对应文件。

`launcher/` 是启动器自己的元数据目录。这里保存状态、profile registry、备份和临时 helper，不应该放用户业务数据或密钥。

Web 管理配置保存在 `launcher/web-server.json`，登录会话保存在 `launcher/web-sessions.json`，访问日志保存在 `launcher/logs/web-server.log`。首次创建时 Web 管理默认开启，监听 `0.0.0.0:9876`，默认访问密码为 `123456`。界面会提示用户修改默认密码；关闭窗口后桌面主进程默认后台常驻，显式退出后 Web 管理停止。

`data/.dock/` 保存 runner 的派生运行清单和运行状态。这里的文件可由 Hermes Dock 重新生成，不是用户业务数据。

## 多 Profile 设计

多 profile 第一版的目标是：在一个 Docker 容器内并行运行多个 Hermes profile gateway worker，让不同 profile 绑定不同的个人微信、企业微信 AI Bot 或飞书 / Lark 应用，并隔离人格、记忆、模型、skills、平台凭据和通道。

运行规则：

- `default` profile 使用 `data/` 根目录，默认进入 profile 列表并参与运行，但允许停用。
- 非 default profile 使用 `data/profiles/<id>/`。
- profile ID 使用路径安全 ASCII slug，例如 `sales`、`support`；中文只作为显示名。
- 每个 enabled profile 如果绑定了完整平台身份，就由 runner 启动对应 gateway。
- enabled 但未绑定平台的 profile 不启动，状态显示为未配置平台。
- 同一个企业微信 Bot、个人微信账号或飞书 App 不能被多个 enabled profile 同时使用。
- 一个 profile 可以同时绑定个人微信、企业微信和飞书，表示同一个助手服务多个入口。
- 平台入口固定归属一个 profile，第一版不做按消息内容跨 profile 路由。
- 配置保存后只写入文件，不自动重建容器；用户手动点击“应用并重建”后统一生效。

隔离边界：

- 按 profile 隔离：`SOUL.md`、`skills/`、`config.yaml`、`.env`、供应商、模型、平台绑定、通道目录、记忆和会话。
- 全局共享：Docker 镜像、端口、容器名、CPU、内存、shm、`docker-compose.override.yaml`。
- 模型供应商和 API Key 默认按 profile 隔离；UI 可以提供显式“复制模型配置到其他 profile”，默认不复制 API Key。
- 平台策略如 `WECOM_DM_POLICY`、`WECOM_GROUP_POLICY`、`WEIXIN_DM_POLICY`、`FEISHU_GROUP_POLICY` 也按 profile 写入各自 `.env`。

runner 设计：

- Compose 只启动一个 Hermes Dock runner，runner 再启动多个 Hermes 子进程。
- runner 统一设置 `HERMES_HOME=/opt/data`，非 default profile 通过 Hermes 原生 `-p <id>` 启动，例如 `hermes -p sales gateway run`。
- Compose 不再依赖全局 `env_file: ./data/.env` 表达 profile 运行态密钥，因为多个 profile 使用同名环境变量。
- runner 为每个 profile 安全加载对应 `.env`，不使用 shell `source`。
- runner 给日志行加 profile 前缀，例如 `[sales] ...`，UI 第一版使用统一容器日志并按前缀过滤。
- runner 对异常退出的 profile 做有限自动重启；连续失败后标记该 profile failed，其他 profile 不受影响。
- 无可运行 profile 时 runner 仍保持容器 running，状态显示无运行 profile。

profile registry 和运行态文件：

- `launcher/profiles.json` 是 Dock 的 profile 事实来源，保存 id、显示名、enabled、创建时间、更新时间和显示顺序，不保存密钥。
- `data/.dock/profiles-runtime.json` 是“应用并重建”时生成的 runner 清单，不需要备份。
- `data/.dock/profile-status.json` 由 runner 写入，Dock 读取展示 profile 进程生命周期状态，不承诺真实平台连接健康。

创建和删除：

- 新建 profile 默认创建干净 profile，不复制密钥、平台账号、记忆、会话或通道目录。
- 第一版可提供“从当前 profile 复制人格和 skills”，不做完整 profile 克隆。
- 非 default profile 可以删除，但删除前必须整体打包备份；`default` profile 不允许删除，只允许停用。
- 删除后允许再次创建同名 profile，但如果残留目录仍存在，不自动复用或覆盖。

暂不做：

- 不做多 Docker 容器或多 Hermes 实例。
- 不为每个 profile 单独暴露 HTTP/API/Dashboard 端口。
- 不做单 profile 启停/重启；第一版统一通过容器重建应用全部变更。
- 不做 Kanban/跨 profile 协作 UI，但保持 Hermes 原生 profile 和 Kanban 机制兼容。
- 不做 profile 导入/导出。

skills 管理和 Skill Hub 安装在桌面端和 Web 端都可用；Web 端不支持打开本机技能目录。

## 数据安全策略

- 默认永不覆盖 `data/` 里的已有文件。
- 首次释放模板时，只创建不存在的文件。
- 修改 `config.yaml`、`.env`、`docker-compose.yaml` 等托管文件前会写入本地备份。
- 修改 profile 的 `config.yaml`、`.env`、`SOUL.md`、`skills/` 或 `launcher/profiles.json` 前也应写入备份。
- 密钥保存在 Hermes 兼容的本地文件中，例如 `data/.env` 和 `data/weixin/accounts/*.json`。
- 启动器状态文件 `launcher/state.json` 不应存放密钥。
- `launcher/profiles.json`、`data/.dock/profiles-runtime.json` 和 `data/.dock/profile-status.json` 都不应存放密钥。
- Web 管理不返回完整 `.env`，不提供 `.env` 明文编辑入口。
- Web 高级编辑只开放当前 profile 的 `config.yaml`、当前 profile 的 `SOUL.md` 和全局 `docker-compose.override.yaml`；保存 Compose 覆盖文件需要输入“确认”。
- Web 管理不提供“恢复出厂设置”。
- “恢复出厂设置”是显式危险操作，会执行 `docker compose down`，删除整个 `~/.hermes-dock`，然后重新释放内置模板。

## Docker Compose

Hermes Dock 接管标准 `~/.hermes-dock/docker-compose.yaml`，用于控制：

- Hermes 镜像版本。
- 网关和控制台端口。
- 控制台账号密码，控制台固定启用。
- 内存、CPU 和 shm 限制。
- `./data:/opt/data` 数据挂载。
- 单 profile 版本使用 `./data/.env` 环境变量注入；多 profile runner 版本不使用全局 `env_file` 表达 profile 密钥。

高级用户如需自定义 Docker 行为，应使用 `~/.hermes-dock/docker-compose.override.yaml`，不要直接依赖手改标准 compose 文件。桌面高级编辑入口可以打开当前 profile 的 `config.yaml` 和 `.env`，用于处理结构化页面尚未覆盖的少量配置；Web 高级编辑不开放 `.env`，只开放当前 profile 的 `config.yaml`、`SOUL.md` 和全局 `docker-compose.override.yaml`。

容器操作对应的 Compose 命令：

- 启动：`docker compose up -d`
- 停止：`docker compose stop`
- 重启：`docker compose restart`
- 重建：`docker compose up -d --force-recreate`

`.env` 变化后，已创建容器不会自动刷新环境变量，需要使用“重建”让新容器拿到最新配置。多 profile 版本中，profile 配置、人设、skills 和平台绑定保存后也需要“应用并重建”才保证运行态生效。

## 模型供应商

供应商配置独立保存在当前 profile 的 `config.yaml` 顶层 `providers` 中，`model.provider` 和辅助模型的 `provider` 字段只引用供应商 ID。启动器保存时会把当前引用供应商的 `base_url`、`api_mode` 和 `api_key` 展开回 `model` / `auxiliary`，兼容 Hermes 当前运行态。

MVP 内置三个供应商实例：

- `dashscope-payg`：DashScope 按量计费，默认模型 `qwen3.7-max`。
- `opencode-go`：OpenCode Go，默认模型 `deepseek-v4-flash`。
- `deepseek`：DeepSeek，默认模型 `deepseek-v4-flash`。

供应商页负责新增、编辑、禁用供应商，以及填写 API Key、接口地址、API 模式和模型列表地址。模型页只选择已配置的供应商和模型名。保存供应商或模型配置时，启动器只把当前主模型和辅助模型实际引用的供应商密钥同步到当前 profile `.env` 的 `DASHSCOPE_API_KEY`、`OPENCODE_GO_API_KEY` 或 `DEEPSEEK_API_KEY`，供对应 profile 运行态读取。

自定义供应商在 UI 中统一保存为 `provider: custom`，适配 OpenAI 兼容或 Anthropic Messages 兼容接口。模型列表不持久化；拉取失败时仍可手动填写模型名。

## 平台绑定

### 个人微信

“平台绑定”页面提供个人微信扫码登录。扫码成功后，启动器会把凭据写入当前 profile 的 `.env` 和 `weixin/accounts/`。多 profile 版本中，保存绑定不自动重建；用户手动应用后运行态生效。

默认策略：

- `WEIXIN_DM_POLICY=open`
- `WEIXIN_GROUP_POLICY=open`

注意：Hermes 当前通过 Tencent iLink Bot API 连接个人微信。普通微信群消息是否能到达，取决于 iLink 侧能力，不完全由 Hermes Dock 控制。

### 企业微信 AI Bot

MVP 只支持企业微信 AI Bot WebSocket。多 profile 版本中，每个 profile 可以绑定一个企业微信 AI Bot，enabled profiles 中 `WECOM_BOT_ID` 必须唯一。默认策略：

- `WECOM_DM_POLICY=open`
- `WECOM_GROUP_POLICY=open`

私聊和群聊策略只支持 `open` 和 `closed`。保存企业微信配置时会清空旧版本的名单字段。

### 飞书 / Lark

MVP 只支持飞书 / Lark WebSocket 模式，用户手动填写 App ID 和 App Secret，不做 webhook 回调配置，也不封装 `hermes gateway setup`。多 profile 版本中，每个 profile 可以绑定一个飞书 / Lark App，enabled profiles 中 `FEISHU_APP_ID` 必须唯一。默认策略：

- `FEISHU_DOMAIN=feishu`
- `FEISHU_CONNECTION_MODE=websocket`
- `FEISHU_GROUP_POLICY=open`

群聊策略只支持 `open` 和 `disabled`，界面显示为“开放”和“关闭”。保存飞书配置时会清空旧版本的名单字段。

## 开发环境

需要：

- Go 1.23+
- pnpm
- Wails v2 CLI
- Docker 和 Docker Compose

常用命令：

```bash
pnpm --dir frontend install
wails generate module
wails dev
```

运行后，应用会管理 `~/.hermes-dock`。不需要再手动设置 `HERMES_DOCK_INSTANCE_ROOT`。

## 项目结构

```text
app.go                 Wails 应用入口和状态聚合
compose.go             Docker Compose 生成和生命周期操作
config.go              Hermes config.yaml 读写、模型供应商和模型列表
env.go                 data/.env 读写和脱敏
weixin.go              个人微信扫码登录 helper 和凭据保存
platforms.go           企业微信、飞书配置、通道和测试消息
templates.go           内置 seed data 释放
paths.go               实例路径和安全路径限制
frontend/src/App.tsx   React 主界面
frontend/src/App.css   界面样式
templates/seed-data/   首次启动释放的干净模板
```

## 构建

开发调试：

```bash
wails dev
```

生成前端绑定：

```bash
wails generate module
```

Go 测试：

```bash
go test ./...
```

前端构建：

```bash
pnpm --dir frontend run build
```

## MVP 范围

当前包含：

- Docker 和 Docker Compose 检测。
- 首次启动从内置干净模板初始化。
- 标准 compose 生成和高级 override 入口。
- 启动、停止、重启、重建、状态和日志。
- 镜像、端口、控制台认证和资源限制编辑。
- 主模型和 auxiliary 模型配置。
- DashScope 按量计费和 DeepSeek 供应商预设。
- 个人微信扫码登录。
- 企业微信 AI Bot WebSocket 配置。
- 飞书 / Lark WebSocket 配置。
- 通道查看、默认通道设置和测试消息发送。
- UI 输出脱敏。
- 写入前本地备份。

当前不做：

- 不安装 Docker。
- 不做系统服务安装。
- 不做多实例管理。
- 当前稳定版本不做多账号平台管理；下一阶段通过单容器多 profile 支持多个平台身份并行。
- 不内置真实运行态、日志、会话、缓存、数据库或用户凭据。
- 不做完整 Hermes 平台配置器，只覆盖 MVP 指定平台。
- 不做内置聊天客户端，聊天仍使用 Hermes 控制台。
- 不在普通导航中提供环境变量编辑器；`.env` 默认由结构化配置和平台绑定流程维护，高级编辑可打开。
