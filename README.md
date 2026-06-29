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
- 支持查看通道目录、设置默认通道、发送测试消息。
- 写入托管文件前自动备份。
- UI 日志和事件会脱敏敏感字段。

## 架构概览

Hermes Dock 的运行模型是“桌面启动器 + 本地 Docker Compose 单实例”：

- Go 后端负责文件读写、备份、Docker Compose 命令、模型列表拉取和平台绑定 helper。
- React 前端负责表单、状态展示、扫码流程、日志输出和通道管理。
- Wails 事件用于推送 Docker 输出、日志行和微信扫码状态。
- Hermes 容器只通过 `./data:/opt/data` 访问用户数据。
- 启动器自己的状态只保存在 `launcher/` 下。

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
    skills/
    weixin/
  launcher/
    state.json
    backups/
    helpers/
```

`data/` 是用户数据。Hermes Dock 默认不会覆盖已有用户数据，只在明确保存配置、绑定平台或执行迁移时写入对应文件。

`launcher/` 是启动器自己的元数据目录。这里保存状态、备份和临时 helper，不应该放用户业务数据。

## 数据安全策略

- 默认永不覆盖 `data/` 里的已有文件。
- 首次释放模板时，只创建不存在的文件。
- 修改 `config.yaml`、`.env`、`docker-compose.yaml` 等托管文件前会写入本地备份。
- 密钥保存在 Hermes 兼容的本地文件中，例如 `data/.env` 和 `data/weixin/accounts/*.json`。
- 启动器状态文件 `launcher/state.json` 不应存放密钥。
- “恢复出厂设置”是显式危险操作，会执行 `docker compose down`，删除整个 `~/.hermes-dock`，然后重新释放内置模板。

## Docker Compose

Hermes Dock 接管标准 `~/.hermes-dock/docker-compose.yaml`，用于控制：

- Hermes 镜像版本。
- 网关和控制台端口。
- 控制台账号密码，控制台固定启用。
- 内存、CPU 和 shm 限制。
- `./data:/opt/data` 数据挂载。
- `./data/.env` 环境变量注入。

高级用户如需自定义 Docker 行为，应使用 `~/.hermes-dock/docker-compose.override.yaml`，不要直接依赖手改标准 compose 文件。高级编辑入口也可以打开 `data/.env`，用于处理结构化页面尚未覆盖的少量配置。

容器操作对应的 Compose 命令：

- 启动：`docker compose up -d`
- 停止：`docker compose stop`
- 重启：`docker compose restart`
- 重建：`docker compose up -d --force-recreate`

`.env` 变化后，已创建容器不会自动刷新环境变量，需要使用“重建”让新容器拿到最新配置。

## 模型供应商

供应商配置独立保存在 `data/config.yaml` 的顶层 `providers` 中，`model.provider` 和辅助模型的 `provider` 字段只引用供应商 ID。启动器保存时会把当前引用供应商的 `base_url`、`api_mode` 和 `api_key` 展开回 `model` / `auxiliary`，兼容 Hermes 当前运行态。

MVP 内置三个供应商实例：

- `dashscope-payg`：DashScope 按量计费，默认模型 `qwen3.7-max`。
- `opencode-go`：OpenCode Go，默认模型 `deepseek-v4-flash`。
- `deepseek`：DeepSeek，默认模型 `deepseek-v4-flash`。

供应商页负责新增、编辑、禁用供应商，以及填写 API Key、接口地址、API 模式和模型列表地址。模型页只选择已配置的供应商和模型名。保存供应商或模型配置时，启动器只把当前主模型和辅助模型实际引用的供应商密钥同步到 `data/.env` 的 `DASHSCOPE_API_KEY`、`OPENCODE_GO_API_KEY` 或 `DEEPSEEK_API_KEY`，供容器运行态读取。

自定义供应商在 UI 中统一保存为 `provider: custom`，适配 OpenAI 兼容或 Anthropic Messages 兼容接口。模型列表不持久化；拉取失败时仍可手动填写模型名。

## 平台绑定

### 个人微信

“平台绑定”页面提供个人微信扫码登录。扫码成功后，启动器会把凭据写入 `data/.env` 和 `data/weixin/accounts/`，并自动重建网关容器，让 Hermes 运行态立即启用 Weixin 通道。

默认策略：

- `WEIXIN_DM_POLICY=open`
- `WEIXIN_GROUP_POLICY=open`

注意：Hermes 当前通过 Tencent iLink Bot API 连接个人微信。普通微信群消息是否能到达，取决于 iLink 侧能力，不完全由 Hermes Dock 控制。

### 企业微信 AI Bot

MVP 只支持企业微信 AI Bot WebSocket。默认策略：

- `WECOM_DM_POLICY=open`
- `WECOM_GROUP_POLICY=open`

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
platforms.go           企业微信配置、通道和测试消息
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
ails generate module
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
- 通道查看、默认通道设置和测试消息发送。
- UI 输出脱敏。
- 写入前本地备份。

当前不做：

- 不安装 Docker。
- 不做系统服务安装。
- 不做多实例管理。
- 不做多账号平台管理。
- 不内置真实运行态、日志、会话、缓存、数据库或用户凭据。
- 不做完整 Hermes 平台配置器，只覆盖 MVP 指定平台。
- 不做内置聊天客户端，聊天仍使用 Hermes 控制台。
- 不在普通导航中提供环境变量编辑器；`.env` 默认由结构化配置和平台绑定流程维护，高级编辑可打开。
