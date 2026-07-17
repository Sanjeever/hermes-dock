# Hermes Dock

Hermes Dock 是一个面向本地单实例 Hermes Agent 的桌面启动器。它基于 Wails 构建，用一个可视化界面管理 `~/.hermes-dock` 下的 Hermes Docker 实例，让不熟悉命令行的新手也能完成初始化、模型配置、平台绑定、启动、停止、重启和重建。

项目目标很明确：只要用户已经安装 Docker，就可以打开 Hermes Dock，完成必要配置，然后启动 Hermes Agent。

## 仓库与发布

- [`sqyl2026/hermes-dock`](https://github.com/sqyl2026/hermes-dock) 是私有源码仓库，用于开发、测试和 GitHub Actions 构建。
- [`sqyl2026/hermes-dock-releases`](https://github.com/sqyl2026/hermes-dock-releases) 是公开二进制发布仓库，提供 Windows、macOS 和 Linux 安装包及 `SHA256SUMS.txt`。
- 应用只从公开发布仓库检查新版本和打开下载链接，不需要访问私有源码仓库。
- 推送版本标签后，私有源码仓库中的 GitHub Actions 完成多平台构建，并将产物发布到公开发布仓库。

## 当前能力

- 首次启动自动创建 `~/.hermes-dock`。
- 释放干净模板到 `~/.hermes-dock/data`，不捆绑当前真实运行态。
- 捆绑 Hermes 内置 skills 快照。
- 生成并接管标准 `docker-compose.yaml`。
- 保留 `docker-compose.override.yaml` 作为高级覆盖入口。
- 可视化管理 Docker 镜像、端口、控制台账号密码、内存、CPU 和共享内存，控制台固定启用。
- 通过模型、部署和平台绑定表单写入必要配置，不向普通用户提供环境变量编辑页。
- 可视化配置主模型和 auxiliary 模型。
- 支持百炼按量计费、百炼 Coding Plan、百炼 Token Plan 团队版、智谱按量计费、智谱 Coding Plan、OpenCode Go、DeepSeek 和 Agnes AI 模型供应商预设。
- 支持通过 API Key 拉取模型列表并选择模型。
- 支持个人微信 Weixin / WeChat Personal 扫码登录。
- 支持企业微信 AI Bot WebSocket 配置。
- 支持飞书 / Lark WebSocket 配置。
- 支持查看通道目录、设置默认通道、发送测试消息。
- 内置 Web 管理界面，随桌面主进程启动，默认局域网可访问。
- 默认集成轻量 Dufs 文件管理，局域网用户可在浏览器中批量上传、整理和下载所有 profile 共享的文件。
- 内置宿主机控制服务，Hermes 可通过 `hostctl` 以当前用户身份静默读写文件、发送通知、操作文本剪贴板、查询进程和端口、截取屏幕、启动应用、执行命令，以及通过窗口、鼠标和键盘完成轻量桌面自动化。
- 写入托管文件前自动备份。
- UI 日志和事件会脱敏敏感字段。

## 架构概览

Hermes Dock 的运行模型是“桌面启动器 + 本地 Docker Compose 单实例”：

- Go 后端负责文件读写、备份、Docker Compose 命令、模型列表拉取和平台绑定 helper。
- React 前端负责表单、状态展示、扫码流程、日志输出和通道管理。
- Wails 事件用于推送 Docker 输出、日志行和微信扫码状态。
- 内置 Web 管理服务运行在桌面主进程内，通过 HTTP RPC 和 WebSocket 复用主要管理能力。
- `website/` 是独立的 React + Vite 官网；构建产物由 Nginx 提供，Node.js 服务提供同域预约演示 API，并通过可配置的外部 SMTP 发送通知。
- Hermes 容器通过 `./data:/opt/data` 访问用户数据，并通过带随机 Token 的 Host Bridge 操作宿主机。
- Dufs sidecar 与 Hermes 挂载同一个共享目录，只负责 Web 文件管理，不承载 Hermes profile。
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
  shared/
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
    host-bridge.token
    dufs/
      config.yaml
    web-server.json
    web-sessions.json
    logs/
      web-server.log
    backups/
    helpers/
      hermes-profile-runner
      install-feishu-deps
      patch-home-channel-prompt
      patch-wecom-filenames
```

`data/` 是用户数据，也是 `default` profile 的 Hermes home。非 default profile 使用 `data/profiles/<id>/`，保持和 Hermes 原生 profile 结构兼容。Hermes Dock 默认不会覆盖已有用户数据，只在明确保存配置、绑定平台或执行迁移时写入对应文件。

`launcher/` 是启动器自己的元数据目录。这里保存状态、profile registry、备份和临时 helper，不应该放用户业务数据或密钥。

`shared/` 是默认共享文件目录，在容器内固定挂载为 `/opt/data/.dock/shared`，由所有 profile 共同读写。用户可以在基础设置中改为其他宿主机绝对路径，目录结构和文件内容由用户自行管理。

Dufs 默认开启并将同一目录挂载为 `/data`，通过局域网 HTTP 端口 `9878` 提供文件管理。默认账号为 `qizhihe`、默认密码为 `123456`，建议首次使用时立即修改；密码只以 SHA-512 crypt 哈希保存在 `launcher/dufs/config.yaml`，不会写入启动器状态或 Compose。Dufs 适合可信局域网，不应直接暴露到公网；需要原生文件系统挂载时可另行使用操作系统自带 SMB 服务。

Web 管理配置保存在 `launcher/web-server.json`，登录会话保存在 `launcher/web-sessions.json`，访问日志保存在 `launcher/logs/web-server.log`。首次创建时 Web 管理默认开启，监听 `0.0.0.0:9876`，默认访问密码为 `123456`；用户可在设置页修改密码。关闭窗口会退出桌面主进程并停止 Web 管理。

启动器会检查公开发布仓库中的稳定版本。发现新版本后可直接点击“立即更新”，启动器会下载对应平台安装包、使用 `SHA256SUMS.txt` 校验、退出并交给独立的 `hermes-dock-updater` 安装，然后重新启动；该过程不会停止 Hermes Docker 容器。静默自动升级默认关闭，只有用户在“设置 → 软件更新”中主动开启后才会注册系统定时任务，关闭时删除任务。Windows 使用当前管理员用户的最高权限计划任务，Linux 使用 systemd system timer，macOS 使用 LaunchDaemon；更新状态和错误保存在 `launcher/update.json`，详细安装日志保存在 `launcher/logs/update.log`。

`data/.dock/` 保存 runner 的派生运行清单和运行状态。宿主机上的这些文件可由 Hermes Dock 重新生成，不是用户业务数据；容器内的 `/opt/data/.dock/shared` 是绑定到外部共享目录的挂载点。

设置页的数据迁移功能会导出 `.hdbackup` 单文件，用于把当前实例迁移到其他设备。备份包含 `data/`、profile registry、Web 管理配置、Dufs 账号密码哈希、标准 Compose 和 Compose override，因此也包含 `.env` 密钥、平台账号凭据和远程访问凭据；共享目录文件、运行日志、Web 登录会话、旧备份和 `data/.dock/` 派生运行态不会写入备份。导出时如果容器正在运行，会先 `docker compose stop`，导出完成后再 `docker compose start` 恢复原运行状态，避免备份到写入中的文件。导入是整实例覆盖流程：先执行 `docker compose down`，再自动生成当前设备的导入前备份，最后恢复备份内容并重新生成标准 `docker-compose.yaml`。

## 多 Profile 设计

当前多 profile 实现会在一个 Docker 容器内并行运行多个 Hermes profile gateway worker，让不同 profile 绑定不同的个人微信、企业微信 AI Bot 或飞书 / Lark 应用，并隔离人格、记忆、模型、skills、平台凭据和通道。

运行规则：

- `default` profile 使用 `data/` 根目录，默认进入 profile 列表并参与运行，但允许停用。
- 非 default profile 使用 `data/profiles/<id>/`。
- profile ID 使用路径安全 ASCII slug，例如 `sales`、`support`；中文只作为显示名。
- 每个 enabled profile 如果绑定了完整平台身份，就由 runner 启动对应 gateway。
- enabled 但未绑定平台的 profile 不启动，状态显示为未配置平台。
- 同一个企业微信 Bot、个人微信账号或飞书 App 不能被多个 enabled profile 同时使用。
- 一个 profile 可以同时绑定个人微信、企业微信和飞书，表示同一个助手服务多个入口。
- 平台入口固定归属一个 profile，第一版不做按消息内容跨 profile 路由。
- 配置保存后只写入文件，不自动应用运行态；用户手动点击“应用配置”后统一生效。

隔离边界：

- 按 profile 隔离：`SOUL.md`、`skills/`、`config.yaml`、`.env`、供应商、模型、平台绑定、通道目录、记忆和会话。
- 全局共享：Docker 镜像、端口、容器名、CPU、内存、shm、`docker-compose.override.yaml`。
- 模型供应商和 API Key 默认按 profile 隔离；UI 可以提供显式“复制模型配置到其他 profile”，默认不复制 API Key。
- 平台策略如 `WECOM_DM_POLICY`、`WECOM_GROUP_POLICY`、`WEIXIN_DM_POLICY`、`FEISHU_GROUP_POLICY` 也按 profile 写入各自 `.env`。

runner 设计：

- Compose 的 Hermes service 只启动一个 Hermes Dock runner，runner 再启动多个 Hermes 子进程；Dufs sidecar 独立管理共享文件。
- runner 统一设置 `HERMES_HOME=/opt/data`，非 default profile 通过 Hermes 原生 `-p <id>` 启动，例如 `hermes -p sales gateway run`。
- Compose 不再依赖全局 `env_file: ./data/.env` 表达 profile 运行态密钥，因为多个 profile 使用同名环境变量。
- runner 为每个 profile 安全加载对应 `.env`，不使用 shell `source`。
- runner 给日志行加 profile 前缀，例如 `[sales] ...`，UI 第一版使用统一容器日志并按前缀过滤。
- runner 对异常退出的 profile 做有限自动重启；连续失败后标记该 profile failed，其他 profile 不受影响。
- 无可运行 profile 时 runner 仍保持容器 running，状态显示无运行 profile。

profile registry 和运行态文件：

- `launcher/profiles.json` 是 Dock 的 profile 事实来源，保存 id、显示名、enabled、创建时间、更新时间和显示顺序，不保存密钥。
- `data/.dock/` 是宿主机 Dock 和容器 runner 共享的可写派生运行态目录，使用 sticky writable 权限且不保存密钥；容器内的 `/opt/data/.dock/shared` 是外部共享目录挂载点，`profiles-runtime.json` 是“应用配置”时生成的 runner 清单，不需要备份。
- `data/.dock/profile-status.json` 由 runner 写入，Dock 读取展示 profile 进程生命周期状态，不承诺真实平台连接健康。

创建和删除：

- 新建 profile 默认创建干净 profile，不复制密钥、平台账号、记忆、会话或通道目录。
- 第一版可提供“从当前 profile 复制人格和 skills”，不做完整 profile 克隆。
- 非 default profile 可以删除；如果容器正在运行，删除前先停止整个容器，再整体打包备份，删除后由用户通过“应用配置”重新启动；`default` profile 不允许删除，只允许停用。
- 删除后允许再次创建同名 profile，但如果残留目录仍存在，不自动复用或覆盖。

暂不做：

- 不做多 Hermes 容器或多 Hermes 实例；Dufs 文件管理 sidecar 是唯一的辅助容器。
- 不为每个 profile 单独暴露 HTTP/API/Dashboard 端口。
- 不做单 profile 启停/重启；第一版统一通过容器重建应用全部变更。
- 不做 Kanban/跨 profile 协作 UI，但保持 Hermes 原生 profile 和 Kanban 机制兼容。
- 不做 profile 导入/导出。

skills 管理、Skill Hub 安装、打开本机技能目录和同步启动器内置最新技能在桌面端和 Web 端都可用。同步内置技能会用启动器模板覆盖当前 profile 的内置技能文件，不删除自定义技能或模板外文件。

## 数据安全策略

- 默认永不覆盖 `data/` 里的已有文件。
- 首次释放模板时，只创建不存在的文件。
- 修改 `config.yaml`、`.env`、`docker-compose.yaml` 等托管文件前会写入本地备份。
- 修改 profile 的 `config.yaml`、`.env`、`SOUL.md`、`skills/` 或 `launcher/profiles.json` 前也应写入备份。
- 密钥保存在 Hermes 兼容的本地文件中，例如 `data/.env` 和 `data/weixin/accounts/*.json`。
- 启动器状态文件 `launcher/state.json` 不应存放密钥。
- Host Bridge Token 单独保存在 `launcher/host-bridge.token`，权限为 `0600`，不写入状态、profile、日志或运行时 manifest。
- `launcher/profiles.json`、`data/.dock/profiles-runtime.json` 和 `data/.dock/profile-status.json` 都不应存放密钥。
- Web 管理与桌面端保持同等管理能力，会返回和编辑当前 profile 的完整环境配置；访问密码是远程管理边界。
- Web 高级编辑与桌面端一致，开放当前 profile 的 `config.yaml`、`.env` 和全局 `docker-compose.override.yaml`；保存 Compose 覆盖文件需要输入“确认”。
- Web 管理提供与桌面端一致的“恢复出厂设置”危险操作。
- “恢复出厂设置”是显式危险操作，会执行 `docker compose down`，删除 `~/.hermes-dock` 中除 `shared/` 外的实例数据，然后重新释放内置模板。

## Docker Compose

Hermes Dock 接管标准 `~/.hermes-dock/docker-compose.yaml`，用于控制：

- Hermes 镜像版本。
- 网关和控制台端口。
- 控制台账号密码，控制台固定启用。
- 内存、CPU 和 shm 限制。
- `./data:/opt/data` 数据挂载。
- 可配置的宿主机共享文件目录固定挂载到 `/opt/data/.dock/shared`，默认使用 `~/.hermes-dock/shared`，由所有 profile 共同读写。
- 默认启用固定版本 `sigoden/dufs:v0.46.0`，将同一共享目录挂载到 `/data` 并通过 `0.0.0.0:9878` 提供轻量 Web 文件管理。
- 数据目录权限由 Hermes 镜像启动脚本定向处理，不在每次应用配置时对整个 `data/` 执行递归 `chown`。
- `launcher/helpers/install-feishu-deps` 挂载到 `/etc/cont-init.d/018-install-feishu-deps`，在 s6 初始化阶段补齐飞书运行依赖；uv 下载缓存持久化到 `data/.dock/uv-cache`。
- `launcher/helpers/patch-home-channel-prompt` 挂载到 `/etc/cont-init.d/019-patch-home-channel-prompt`，在固定 Hermes 镜像启动时关闭未设置 Home Channel 的首次对话提示，不影响 `/sethome` 和实际投递校验。
- `launcher/helpers/hostctl` 挂载到 `/usr/local/bin/hostctl`，通过 `host.docker.internal:9877` 调用桌面主进程内的 Host Bridge。
- 单 profile 版本使用 `./data/.env` 环境变量注入；多 profile runner 版本不使用全局 `env_file` 表达 profile 密钥。

资源配额默认值按 Docker daemon 可用资源计算，不直接读取物理机总资源：

- 内存限制：`max(floor(Docker MemTotal / GiB) - 2, 1)G`，给系统保留 2G。
- CPU 限制：使用 Docker `NCPU` 全量，格式化为一位小数，例如 `8.0`。
- 读取 Docker 失败时使用固定 fallback：`4G` / `2.0`。
- 只在首次初始化或配置字段缺失时填充，不覆盖用户已保存的资源配额；旧用户已有 `4G` / `2.0` 也保持不变。
- 设置页提供“使用推荐值”显式按当前 Docker 可用资源重算，点击后仍需保存设置。
- `shm_size` 继续默认 `1g`，不随宿主机动态计算。

高级用户如需自定义 Docker 行为，应使用 `~/.hermes-dock/docker-compose.override.yaml`，不要直接依赖手改标准 compose 文件。桌面和 Web 高级编辑入口都可以打开当前 profile 的 `config.yaml`、`.env` 和全局 `docker-compose.override.yaml`，用于处理结构化页面尚未覆盖的少量配置。

宿主机控制默认开启，不做逐次审批。Host Bridge 只接受持有随机 Token 的请求，所有操作以启动 Hermes Dock 的当前用户身份执行，不自动提权。结构化能力包括最大 16 MiB 的文件读写、最大 1 MiB 的文本剪贴板、进程和网络连接查询、系统通知、默认应用打开、应用启动，以及最大 25 MiB 的多显示器 PNG 截图；Shell 命令默认超时 120 秒、最长 1800 秒，stdout 和 stderr 分别最多返回 1 MiB。

轻量桌面自动化通过 `hostctl window`、`hostctl mouse` 和 `hostctl keyboard` 提供。鼠标使用指定显示器截图内的局部像素坐标；点击、拖拽、滚动和键盘输入必须携带预期前台窗口 ID，窗口变化时立即失败。物理键鼠由所有 profile 共享，Host Bridge 使用 30 秒短租约串行化不同 profile 的操作。Windows 使用原生窗口和输入 API，macOS 需要“辅助功能”和“屏幕录制”权限；Linux 仅在 X11 且已安装 `xdotool` 时适度支持，不自动安装依赖或绕过 Wayland 限制。

关闭 Hermes Dock 后 Host Bridge 随主进程停止。该能力会把来自模型和平台消息的指令转化为宿主机操作，设置页会持续显示风险提示。

容器操作对应的 Compose 命令：

- 启动：`docker compose up -d`
- 停止：`docker compose stop`
- 重启：`docker compose restart hermes`
- 重建：`docker compose up -d --force-recreate --remove-orphans hermes`

“应用配置”会比较 Hermes、Dufs 和 override 的当前指纹与上次成功应用的指纹。只修改 profile 的 `.env`、`config.yaml`、`SOUL.md`、skills、平台绑定或启停状态时，复用现有容器并重启 `hermes` 服务；镜像、端口、资源、代理或 Compose override 变化时才重新创建 Hermes 容器。只修改 Dufs 开关、端口或共享账号时仅更新 Dufs，不重启 Hermes。涉及 Hermes 的两种路径都会等待当前 runtime generation 上报就绪后再标记应用成功。

## 模型供应商

供应商配置独立保存在当前 profile 的 `config.yaml` 顶层 `providers` 中，`model.provider` 和辅助模型的 `provider` 字段只引用供应商 ID。启动器保存时会把当前引用供应商的 `base_url`、`api_mode` 和 `api_key` 展开回 `model` / `auxiliary`，兼容 Hermes 当前运行态。

MVP 内置八个供应商实例：

- `dashscope-payg`：百炼按量计费，默认模型 `qwen3.7-max`。
- `bailian-coding-plan`：百炼 Coding Plan，默认模型 `qwen3.7-max`。
- `bailian-token-plan-team`：百炼 Token Plan 团队版，使用 OpenAI 兼容接口，模型名手动填写。
- `zhipu-payg`：智谱按量计费，默认模型 `glm-5.2`。
- `zhipu-coding-plan`：智谱 Coding Plan，默认模型 `glm-5.2`。
- `opencode-go`：OpenCode Go，默认模型 `deepseek-v4-flash`。
- `deepseek`：DeepSeek，默认模型 `deepseek-v4-flash`。
- `agnes`：Agnes AI，默认模型 `agnes-2.0-flash`。

供应商页负责新增、编辑、禁用供应商，以及填写 API Key、接口地址、API 模式和模型列表地址。模型页只选择已配置的供应商和模型名。保存供应商或模型配置时，启动器只把当前主模型和辅助模型实际引用的供应商密钥同步到当前 profile `.env` 的 `DASHSCOPE_API_KEY`、`ZHIPU_API_KEY`、`OPENCODE_GO_API_KEY`、`DEEPSEEK_API_KEY` 或 `AGNES_API_KEY`，供对应 profile 运行态读取。

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

MVP 只支持飞书 / Lark WebSocket 模式。默认通过扫码自动创建并绑定机器人；Dock 会自动识别飞书或 Lark 区域，并在扫码成功后保存凭据。已有应用可展开“使用已有应用（高级）”手动填写 App ID 和 App Secret，不做 webhook 回调配置。多 profile 版本中，每个 profile 可以绑定一个飞书 / Lark App，enabled profiles 中 `FEISHU_APP_ID` 必须唯一。默认策略：

- `FEISHU_DOMAIN=feishu`
- `FEISHU_CONNECTION_MODE=websocket`
- `FEISHU_ALLOW_ALL_USERS=true`
- `FEISHU_GROUP_POLICY=open`

群聊策略只支持 `open` 和 `disabled`，界面显示为“开放”和“关闭”。重新扫码创建机器人会要求确认，只有扫码成功才替换当前 profile 的绑定；旧飞书应用不会被 Dock 自动删除。保存飞书配置时会清空旧版本的名单字段。

Hermes Dock 会在容器初始化阶段通过包元数据快速检查 `lark-oapi==1.5.3` 和 `qrcode==7.4.2`。缺少时，`/etc/cont-init.d/018-install-feishu-deps` 会使用 Compose 中配置的 Python 镜像源安装到 `/opt/hermes/.venv`，安装后再次执行 import 验证；uv 下载缓存保存在 `data/.dock/uv-cache`，容器重新创建后可以复用。该流程只补齐运行依赖，不读取或输出飞书 App Secret。

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

## Agent 技能

本仓库当前包含以下开发期 Agent 技能，文件位于 `.agents/skills/`，来源记录在 `skills-lock.json`。这些技能用于辅助维护 Hermes Dock 本身，不同于 Hermes 运行时放在 `templates/seed-data/skills/` 里的 Agent skills。

### 开发质量技能

本项目通过 [`npx skills add`](https://github.com/vercel-labs/skills) 安装了 [obra/superpowers](https://github.com/obra/superpowers) 的两个开发质量技能。`systematic-debugging` 用于在修改代码前查明问题根因，`requesting-code-review` 用于在实现完成后通过独立审查发现遗漏；两者分别覆盖“如何修”和“如何验”，可以在同一任务中顺序使用。

| 技能 | 用途 | 适合在 Hermes Dock 中使用的场景 |
| --- | --- | --- |
| `systematic-debugging` | 系统化定位根因并验证修复 | 遇到 Go 测试失败、Wails / React 构建异常、Docker Compose 行为不符合预期、平台绑定故障、性能问题或偶发错误时使用。技能要求先阅读错误、稳定复现、检查近期变更并沿调用链收集证据，再提出单一假设并做最小验证，不能先猜测修复或同时尝试多个改动。 |
| `requesting-code-review` | 让独立 Agent 审查已完成的实现 | 完成较大功能、复杂缺陷修复或准备合并到 `main` 前使用。技能会向审查 Agent 提供实现说明、需求和明确的 Git 范围，要求只读检查需求符合度、代码质量、架构、安全、测试和上线准备，并将问题分为 Critical、Important 和 Minor。 |

#### `systematic-debugging`

路径：`.agents/skills/systematic-debugging/`。该技能把调试分为根因调查、模式对比、假设验证和修复实现四个阶段；必须完成前一阶段后才能进入下一阶段。对于 Hermes Dock 这类横跨 Go 后端、Wails 绑定、React 界面、Docker Compose 和容器内 runner 的系统，它尤其强调在组件边界检查输入、输出、配置传播和运行状态，从证据中确定真正发生故障的层级。

适合使用的场景：

- Docker 容器状态与界面展示不一致，需要追踪 Compose、Go 状态聚合和前端事件之间的数据流。
- profile 的 `.env`、runtime manifest 或平台绑定保存成功，但重启后的 gateway 没有获得预期配置。
- 测试偶发失败、依赖固定延时才能通过，或连续尝试多个修复后不断出现新症状。
- 构建、发布、自动升级或官网部署经过多个进程和环境，需要先确认故障发生在哪个边界。

技能目录还包含根因回溯、分层校验和基于条件等待的说明，以及用于定位污染测试的 `find-polluter.sh`。查明根因后，应先建立能复现问题的失败用例，只实施针对根因的最小修复，再验证原问题和相关测试；如果三次修复尝试仍失败，应暂停继续打补丁并重新审视架构假设。

示例提示词：

```text
使用 systematic-debugging 调查“应用配置后某个 enabled profile 仍使用旧 API Key”的问题。先稳定复现，检查 profile .env、runtime manifest、runner 环境解析和 gateway 子进程之间的数据流，提出并验证单一根因假设；在找到根因前不要修改代码。
```

#### `requesting-code-review`

路径：`.agents/skills/requesting-code-review/`。该技能要求把实现摘要、原始需求、`BASE_SHA` 和 `HEAD_SHA` 交给独立审查 Agent，让审查者直接读取指定范围的 diff，而不是依赖实现过程中的对话历史。随附的 `code-reviewer.md` 模板规定审查过程保持只读，并要求每个问题提供文件行号、影响和修复方向，最后明确判断是否可以合并。

适合使用的场景：

- 完成多 profile、备份迁移、Host Bridge、自动升级等涉及数据安全或跨组件契约的功能后。
- 修复复杂故障后，确认改动解决的是根因，没有引入静默降级、数据覆盖、密钥泄露或兼容性问题。
- 准备合并前，核对实现是否完整满足 README、AGENTS.md、计划或用户需求，并检查测试是否覆盖真实行为和关键边界。
- 开始较大重构前先审查现有基线，或在实现受阻时引入不带既有思路偏见的独立视角。

收到审查结果后，应立即处理 Critical 问题，并在继续或合并前处理 Important 问题；Minor 问题可以记录后续处理。如果不同意审查意见，应以代码、测试或明确的项目约束说明原因，而不是直接忽略。该技能依赖当前 Agent 环境支持派发独立审查 Agent。

示例提示词：

```text
使用 requesting-code-review 审查这次多 profile runner 改动。需求以 AGENTS.md 的“多 Profile 设计约定”和“数据安全规则”为准，审查范围为 origin/main 到当前 HEAD；重点检查 profile 隔离、平台身份唯一性、密钥是否进入派生运行态、异常重启边界和测试覆盖。审查过程只读，不要修改工作树。
```

### 营销文案技能

本项目通过 [`npx skills add`](https://github.com/vercel-labs/skills) 安装了 [coreyhaines31/marketingskills](https://github.com/coreyhaines31/marketingskills) 的两个文案技能。它们适合官网、产品介绍页、功能页、定价页和发布页等需要说明价值并引导行动的内容；不应用于日常的技术文档、界面字段文案或与转化无关的代码注释。

| 技能 | 用途 | 适合在 Hermes Dock 中使用的场景 |
| --- | --- | --- |
| `copywriting` | 从零起草或重写营销文案 | 为官网、产品介绍页、版本发布页或功能专题页编写标题、副标题、价值主张、正文和行动按钮。使用时先提供页面目标、目标读者、核心问题、产品差异、可核实的证明材料和流量来源；技能会优先清晰、具体的用户收益和可执行的 CTA，不应编造数据、客户评价或案例。 |
| `copy-editing` | 审校、润色和刷新已有营销文案 | 改进已有官网、发布页、功能页或产品手册文案，同时保留原有事实和核心信息。它按“清晰度、语气、用户收益、证据、具体性、情感、行动风险”七个维度逐轮检查，适合处理文案生硬、冗长、卖点不清、证据不足或内容过时等问题。 |

两个技能的分工很明确：需要新文案时先用 `copywriting`；已有初稿或线上内容需要改进时用 `copy-editing`。前者完成初稿后，可再用后者进行审校。若仓库存在 `.agents/product-marketing.md`，两个技能都会先读取它，确保产出符合已有的产品定位、品牌语气和用户语言。

示例提示词：

```text
使用 copywriting 为 Hermes Dock 的产品介绍页撰写中文文案。目标是让已安装 Docker、但不熟悉命令行的个人用户开始配置本地 Hermes Agent；主要 CTA 是“下载并开始配置”。突出可视化管理、个人微信/企业微信/飞书接入和本地数据控制。不要虚构用户数量、性能数据或客户评价；为标题和 CTA 各提供 3 个备选方案。
```

```text
使用 copy-editing 审校以下 Hermes Dock 版本发布页文案。保留所有已有事实，不新增未经证实的数据；按七轮检查指出问题、给出可替换的具体修改，并重点检查中文表述是否清晰、功能是否解释为用户收益、CTA 是否明确。
```

### Taste Skill 系列

本项目通过 [`npx skills add`](https://github.com/vercel-labs/skills) 安装了 [Leonxlnx/taste-skill](https://github.com/Leonxlnx/taste-skill) 的 13 个开发期技能。该系列的目标是让 Agent 先理解页面目标、受众和已有品牌资产，再选择有意图的排版、字体、留白和动效，避免把渐变、玻璃效果、三列等高卡片等模板化视觉模式当作默认答案。

其中“实现类”技能会指导代码或设计评审；“图像类”技能只生成视觉参考图，之后仍需由实现类技能或开发者将参考图落到 React/CSS。它们是可组合的指导，不要求每次前端改动全部启用。Hermes Dock 是以信息密度和可操作性为核心的桌面管理工具：`ui-ux-pro-max` 仍是通用界面、表单、可访问性和响应式改动的首选；Taste Skill 系列应按场景补充使用，不能为了视觉风格牺牲中文可读性、状态反馈或实际操作路径。

| 技能 | 类型 | 适合在 Hermes Dock 中使用的场景 |
| --- | --- | --- |
| `design-taste-frontend` | 实现与设计规范 | 默认的 Taste Skill v2（实验性）。适合需要显著调整主页、Web 管理页或产品介绍页时：先输出一行设计判断，再按页面语境设置布局变化、动效强度和信息密度三个尺度；包含重设计审计、依赖核对、性能与无障碍预检。它主要面向落地页、作品集和重设计，不应直接套用于高密度数据表。 |
| `design-taste-frontend-v1` | 实现与设计规范 | 原始 v1 规则，保留用于需要其既有行为的工作流。默认不要与 v2 同时作为主规则；只有 v2 的实验性调整确实影响现有产出时才显式选用。 |
| `gpt-taste` | 实现与动效规范 | 面向 GPT/Codex 的更激进变体，强调布局随机性、AIDA 页面结构、宽幅标题、无缝 bento 网格与 GSAP 滚动编排。适合独立的官网、发布页或活动页，不适合常规设置页；使用前须确认项目已有的动画依赖和性能预算。 |
| `image-to-code` | 图像优先的实现工作流 | 视觉质量特别重要的网站任务：先生成足量、按区块拆分的设计图，分析字体、间距、媒体框架和组件细节，再实现并防止代码逐渐偏离参考图。适合新的官网或 Web 管理视觉大改，不用于仅修改一个表单字段。 |
| `redesign-existing-projects` | 现有界面审计与重设计 | 改造现有 `frontend/src/App.tsx` / `App.css` 前先审计字体、颜色、布局、交互状态、图标和代码质量，再按优先级修复。适合“界面显得普通或拥挤，但功能必须保持”的任务。 |
| `high-end-visual-design` | 高端视觉与动效规范 | 提供柔和、克制、有留白的高端视觉方向，以及材质、微交互和性能约束。适合需要提升品牌感的欢迎页、空状态或 Web 管理登录页；不应覆盖 Dock 工具页现有清晰、克制的操作导向。 |
| `minimalist-ui` | 风格规范 | 暖色单色系、编辑感排版、扁平 bento 布局和低干扰微动效。适合追求安静、专业的配置或阅读界面；明确避免渐变、重阴影和多余装饰。 |
| `industrial-brutalist-ui` | 风格规范 | 瑞士印刷与战术终端混合的工业粗野风格，强调网格、极端字号对比、遥测信息和纹理。适合可选的诊断/日志演示页面或独立实验页面，不应作为普通新手设置流程的默认风格。 |
| `stitch-design-taste` | 设计系统文档生成 | 为 Google Stitch 生成可执行的 `DESIGN.md`：统一色彩、字体、组件、响应式、动效和反模式。只有在用 Stitch 生成多屏设计稿或希望将视觉规范交给其他 Agent 时使用；它生成规范，不直接改应用代码。 |
| `full-output-enforcement` | 输出完整性约束 | 当任务明确要求完整文件、多个组件或大量重复但不能省略的代码时启用，禁止用占位符、半成品骨架或“其余类似”替代实现。普通的小型修复无需启用，以免不必要地拉长输出。 |
| `imagegen-frontend-web` | 图像类 | 为官网、落地页和营销页面生成视觉参考。关键规则是**每个区块单独生成一张横向图**，保持全页色彩一致但让构图、CTA 与区块节奏有变化；不输出前端代码。 |
| `imagegen-frontend-mobile` | 图像类 | 为 iOS、Android 或跨平台应用生成高保真屏幕和流程参考，强调安全区、导航、文字可读性、屏间一致性与设备框。适合未来移动端产品概念，不适合替代当前 Wails 桌面界面的实现。 |
| `brandkit` | 图像类 | 生成品牌手册板、标志方向、配色、字体和应用 mockup 的整套视觉世界。适合 Hermes Dock 需要重新定义品牌资产或制作设计提案时使用；不应用它直接产出最终 SVG/图标源文件。 |

建议的组合方式：现有界面升级先用 `redesign-existing-projects` 审计，再按需要加入 `ui-ux-pro-max` 或一个明确的风格技能；视觉先行的官网任务使用 `imagegen-frontend-web` 或 `brandkit` 生成参考，再用 `image-to-code` 实现；需要完整交付多个文件时再叠加 `full-output-enforcement`。任何引入新依赖、GSAP 或图片资源的实现，都必须先检查当前 `frontend/package.json` 和项目性能、无障碍约束。

#### `design-taste-frontend`

路径：`.agents/skills/design-taste-frontend/`。这是 Taste Skill 的默认 v2（实验性）实现技能。它要求先判断页面类型、受众、品牌资产和约束，再以布局变化、动效强度、信息密度三个尺度组织设计，并在交付前检查无障碍、性能、暗色模式和常见 AI 视觉套路。

适合使用的场景：

- 为 Hermes Dock 增加独立的产品介绍页、发布说明页或 Web 管理登录页，并需要明确的视觉方向。
- 对现有 Web 管理页做较大范围改版，同时保留已有品牌资产、功能和 Wails / RPC 契约。
- 需要为一项新页面决定是否采用成熟设计系统，或采用自定义但受约束的视觉语言。

示例提示词：

```text
使用 design-taste-frontend 重设计 Hermes Dock 的 Web 管理登录页。先给出一行 Design Read，再以信任优先、低动效、低密度为方向实现；保留访问密码登录和现有 API，不使用营销型渐变或大面积玻璃效果。
```

#### `design-taste-frontend-v1`

路径：`.agents/skills/design-taste-frontend-v1/`。这是保留的原始 Taste Skill v1，提供固定的高变化度、适中动效、低密度基线，以及较早期的前端架构和反模板化规则。它不是 v2 的叠加层。

适合使用的场景：

- 已有提示词、评审标准或生成流程明确依赖 v1 的行为，需要避免 v2 实验性规则改变结果。
- 需要复现过去用 v1 产出的独立网页风格，便于视觉一致性对比。

示例提示词：

```text
使用 design-taste-frontend-v1 为 Hermes Dock 的版本发布页制作一个与既有 v1 页面一致的视觉方案。只修改发布页相关组件，先检查 package.json 中是否已有图标和动效依赖。
```

#### `gpt-taste`

路径：`.agents/skills/gpt-taste/`。这是针对 GPT/Codex 的高强度创意方向，要求用真正的随机变化打破重复布局，并强调 AIDA 叙事、两行以内的宽幅标题、无缝 bento 和 GSAP 的滚动触发动画。

适合使用的场景：

- 制作 Hermes Dock 官网、重大版本发布页、产品演示页或活动页。
- 有明确动效预算、可接受较高视觉变化度，并希望由 GSAP 驱动滚动分段、固定或堆叠叙事的页面。

不适合：设置表单、日志页、profile 管理等高频工具界面，也不应在未确认依赖和 `prefers-reduced-motion` 策略时直接引入 GSAP。

示例提示词：

```text
使用 gpt-taste 为 Hermes Dock 1.0 发布页设计并实现 AIDA 结构。标题保持两行以内，动效只服务于版本功能叙事；先确认 GSAP 是否已安装，并为 prefers-reduced-motion 提供静态体验。
```

#### `image-to-code`

路径：`.agents/skills/image-to-code/`。该技能规定“先图像、后实现”的流程：生成足量且可读的区块级设计图，分析其排版、间距、颜色、媒体和组件，再把分析结果落实到代码，避免从一张压缩总览图猜测整页细节。

适合使用的场景：

- 新建 Hermes Dock 官网或需要大幅换肤的 Web 管理页，且视觉质量比快速套模板更重要。
- 已有设计图或需要生成设计图，希望把实现与参考图逐区块对齐。
- 需要在小笔记本屏幕的首屏中验证主标题、CTA 和关键状态是否清楚可见。

示例提示词：

```text
使用 image-to-code 为 Hermes Dock 官网完成“生成设计图、逐图分析、再实现”的流程。为 hero、profile 概览、平台接入和安全说明分别生成参考图，不裁剪旧图代替新区块；实现时保持中文文案可读。
```

#### `redesign-existing-projects`

路径：`.agents/skills/redesign-existing-projects/`。该技能面向已有代码库，先从排版、表面、布局、交互状态、内容、图标和代码质量做审计，再按影响程度应用升级技巧，避免在不了解现状时整体推倒重来。

适合使用的场景：

- 用户反馈主界面“显得普通、拥挤或不够专业”，但功能、Wails 调用和状态管理必须保持不变。
- 改进 profile 总览、模型供应商或平台绑定页面的层级、错误态、空状态、加载态和按钮反馈。
- 对 `frontend/src/App.tsx` 与 `frontend/src/App.css` 做以问题为导向的视觉重构。

示例提示词：

```text
使用 redesign-existing-projects 审计 Hermes Dock 的 profile 总览。先列出排版、布局、状态反馈和图标四类问题，按优先级提出修改并实施；不得改变任何 Wails 调用、profile 数据结构或保存行为。
```

#### `high-end-visual-design`

路径：`.agents/skills/high-end-visual-design/`。该技能提供柔和对比、精致留白、材质感、嵌套层次和弹簧微交互的高端视觉方向，同时限制会造成性能问题的实现方式。

适合使用的场景：

- 为欢迎页、首次初始化完成页、空状态或 Web 管理登录页增加克制的品牌质感。
- 需要从明确的氛围、纹理和布局原型中选择一个设计方向，而不是让组件都长成同一种卡片。
- 设计可感知但不干扰操作的按钮、导航和进入动画。

示例提示词：

```text
使用 high-end-visual-design 优化 Hermes Dock 的首次初始化完成页：视觉应安静、可信、有留白，突出“开始配置”和“查看运行状态”两个动作；不要使用霓虹渐变、自动播放的大型动画或营销口号。
```

#### `minimalist-ui`

路径：`.agents/skills/minimalist-ui/`。该技能用于偏编辑感的实用极简界面：暖色中性色配合少量低饱和点缀色、清楚的文字层级、平面分组与轻微交互，明确限制渐变、重阴影和无意义装饰。

适合使用的场景：

- 改进部署配置、模型配置、高级编辑或说明较多的页面，使长中文表单和帮助文字保持安静易读。
- 为配置摘要、未应用变更提示或只读详情创建清晰的平面信息层次。

示例提示词：

```text
使用 minimalist-ui 重构 Hermes Dock 的模型供应商详情区。采用暖色中性基底和单一低饱和强调色，重点改善长中文标签、API Key 状态、保存提示和错误信息的阅读顺序；不改变字段和保存逻辑。
```

#### `industrial-brutalist-ui`

路径：`.agents/skills/industrial-brutalist-ui/`。该技能将瑞士工业印刷与战术遥测终端结合，提供宏观标题、等宽微型数据、硬边网格、有限色彩和颗粒/扫描线等质感规则。

适合使用的场景：

- 为容器日志、诊断快照或开发者演示制作独立的只读实验视图。
- 为需要突出进程状态、时间戳、profile 前缀和资源数据的可视化原型定义一致的终端式语言。

不适合：普通新手用户的配置、绑定和危险操作页面；这些流程应优先清晰度、低认知负担和可访问性。

示例提示词：

```text
使用 industrial-brutalist-ui 为 Hermes Dock 设计一个只读的容器诊断原型：突出 profile 前缀、退出状态、CPU 和内存数据，使用等宽数据字体和有限的状态色；不要替换现有的常规设置界面。
```

#### `stitch-design-taste`

路径：`.agents/skills/stitch-design-taste/`。该技能把设计意图转为 Google Stitch 可理解的 `DESIGN.md`，定义视觉氛围、色彩角色、排版、组件行为、响应式规则、动效和明确禁止的反模式，作为多屏设计的一份事实来源。

适合使用的场景：

- 使用 Google Stitch 探索 Hermes Dock 的 Web 管理页或未来移动端概念，并需要多张页面保持同一设计系统。
- 需要把设计决策交给多个 Agent 或设计工具执行，避免只靠零散提示词造成视觉漂移。

示例提示词：

```text
使用 stitch-design-taste 为 Hermes Dock Web 管理创建 DESIGN.md。设计系统应优先中文可读性、工具型密度、访问密码登录与危险操作辨识度；包含颜色、字体、表单状态、响应式和禁止使用的营销化视觉模式。
```

#### `full-output-enforcement`

路径：`.agents/skills/full-output-enforcement/`。该技能是交付完整性约束：先锁定用户要求的文件或组件数量，禁止用占位符、截断片段或“其余相同”替代真实内容；接近输出上限时要求在明确边界暂停并标明续写位置。

适合使用的场景：

- 用户要求一次性完整生成多个 React 组件、完整 CSS 文件、所有状态分支或所有平台表单。
- 文档任务明确要求覆盖每一个技能、每一项配置或每一个目录，而不能只给代表性示例。
- 代码审查后需要交付全部已确认的修复文件，而非只展示首个修复。

示例提示词：

```text
使用 full-output-enforcement 完整实现 Hermes Dock 的平台绑定表单重构。交付范围包括个人微信、企业微信、飞书三个表单及其加载、错误、空状态；不要使用 TODO、占位组件或省略重复字段。
```

#### `imagegen-frontend-web`

路径：`.agents/skills/imagegen-frontend-web/`。这是只生成图像的网页艺术指导技能。它要求一个页面的每个区块各自生成一张横向参考图，并以统一的色彩和叙事主线连接这些图，同时让构图、CTA、背景和区块节奏避免机械重复。

适合使用的场景：

- 在实现前为 Hermes Dock 官网、产品发布页或功能专题页制作可供讨论的区块级视觉稿。
- 需要比较多种英雄区构图、功能展示方式或 CTA 方向，但还不应改动生产代码。

示例提示词：

```text
使用 imagegen-frontend-web 为 Hermes Dock 官网生成 6 张独立横向参考图，分别对应 hero、Docker 状态、profile 管理、平台绑定、数据安全和下载 CTA。保持一套克制的中文开发者工具视觉系统，不在一张大图中拼接多个区块。
```

#### `imagegen-frontend-mobile`

路径：`.agents/skills/imagegen-frontend-mobile/`。这是只生成图像的移动端屏幕和流程指导技能，涵盖 iOS、Android 与跨平台方向；它重视安全区、导航方式、文字大小、屏幕间一致性、真实流程和设备框呈现。

适合使用的场景：

- 探索 Hermes Dock 未来的移动端远程管理概念，例如容器状态查看、扫码登录和 profile 详情。
- 在编码前验证一组移动流程是否能让用户看清关键状态、危险操作和下一步动作。

示例提示词：

```text
使用 imagegen-frontend-mobile 为 Hermes Dock 的移动端远程管理概念生成 5 个 iOS 风格屏幕：容器总览、profile 列表、profile 详情、二维码登录和恢复出厂设置确认。保持同一配色和导航逻辑，文字需在手机尺寸下可读。
```

#### `brandkit`

路径：`.agents/skills/brandkit/`。这是只生成图像的品牌视觉系统技能，先建立品牌隐喻和标志概念，再在同一展示板中组织标志构造、配色、字体、数字界面、实体应用、图像方向和系统细节。

适合使用的场景：

- Hermes Dock 需要确定名称、图标、色彩、字体与产品视觉的统一方向，或为重新定位制作品牌提案。
- 在开始制作官网、Web 管理登录页和应用图标前，先评审品牌系统是否足以跨多个触点使用。

不适合：直接生成可提交的最终 SVG、应用图标源文件或替换现有品牌资产；生成图像应作为设计方向，最终资产仍需单独制作和审阅。

示例提示词：

```text
使用 brandkit 为 Hermes Dock 生成一张 3×3 品牌系统提案板。核心隐喻是“可靠的本地停靠与多 profile 调度”，需要包含标志方向、构造图、深浅色配色、中文/英文排版、桌面应用界面和开发者工具应用示例；避免加密货币或赛博朋克刻板印象。
```

### `ui-ux-pro-max`

路径：`.agents/skills/ui-ux-pro-max/`。该技能用于界面设计、交互改进、视觉质量检查、无障碍和响应式评审，适合维护 Hermes Dock 的 React / Wails 界面。

适合使用的场景：

- 设计或重构 `frontend/src/App.tsx`、`frontend/src/App.css` 中的页面结构、导航、表单、按钮、卡片、表格和状态展示。
- 改进多 profile 总览、平台绑定、模型供应商、Web 管理、高级编辑等界面的信息层级和交互流程。
- 检查移动端窗口、窄屏桌面、深色/浅色主题、长中文文案、加载态、错误态和空状态。
- 评审无障碍、键盘导航、焦点状态、触控目标、颜色对比度和响应式布局。
- 为数据状态、容器日志、profile 状态或未来图表选择合适的展示方式。

示例提示词：

```text
使用 ui-ux-pro-max 检查 Hermes Dock 的首页 profile 总览，重点看中文桌面工具的密度、信息层级、按钮布局和移动端适配。
```

```text
使用 ui-ux-pro-max 重构平台绑定页的交互文案和错误态，但保持当前 React 组件结构和 Wails 调用不变。
```

```text
使用 ui-ux-pro-max 评审 frontend/src/App.css，找出影响可读性、焦点可见性、触控目标和响应式布局的问题，只输出按严重程度排序的改进建议。
```

```text
使用 ui-ux-pro-max 为多 profile 管理页设计一个克制、工具型、适合中国大陆新手用户的布局方案，避免营销型首页和大段说明文字。
```

```text
使用 ui-ux-pro-max 优化模型供应商配置表单，要求 API Key 不暴露、错误提示靠近字段、保存和测试状态清晰。
```

该技能自带检索脚本。需要生成或查询设计建议时，可以从技能目录运行：

```bash
uv run --no-project python .agents/skills/ui-ux-pro-max/scripts/search.py "desktop launcher admin dashboard" --design-system -p "Hermes Dock"
uv run --no-project python .agents/skills/ui-ux-pro-max/scripts/search.py "accessibility responsive forms" --domain ux
uv run --no-project python .agents/skills/ui-ux-pro-max/scripts/search.py "react form state loading error" --stack react
```

### `vercel-react-best-practices`

路径：`.agents/skills/vercel-react-best-practices/`。该技能来自 Vercel Engineering 的 React / Next.js 性能优化实践，用于编写、评审和重构 React 代码时检查瀑布请求、bundle 体积、重渲染、客户端数据读取和 JavaScript 性能问题。

适合使用的场景：

- 新增或重构 `frontend/src/App.tsx` 中的 React 组件、hooks、状态派生和事件处理。
- 检查保存、测试、扫码、日志流、WebSocket 事件等异步流程是否存在不必要的串行等待。
- 优化大列表、日志输出、profile 状态聚合、表单联动和搜索过滤等可能频繁重渲染的界面。
- 评审导入方式、懒加载边界、重型依赖和 bundle 体积。
- 在不改变 Wails Go 方法契约的前提下，改进前端性能和可维护性。

示例提示词：

```text
使用 vercel-react-best-practices 评审 frontend/src/App.tsx，重点检查不必要的重渲染、派生 state、effect 依赖和异步瀑布。
```

```text
使用 vercel-react-best-practices 优化容器日志视图，避免日志持续追加时导致整页频繁重渲染。
```

```text
使用 vercel-react-best-practices 检查平台绑定页的扫码和测试消息流程，找出可以并行化或移出 effect 的逻辑。
```

```text
使用 vercel-react-best-practices 重构模型供应商表单，保持行为不变，减少无效 memo、复杂 effect 和非必要对象依赖。
```

更新单个技能：

```bash
npx -y skills update copywriting --project --yes
npx -y skills update copy-editing --project --yes
npx -y skills update systematic-debugging --project --yes
npx -y skills update requesting-code-review --project --yes
npx -y skills update ui-ux-pro-max --project --yes
npx -y skills update vercel-react-best-practices --project --yes
```

更新当前项目的全部技能：

```bash
npx -y skills update --project --yes
```

如果本地改过 `.agents/skills/` 下的技能文件，更新前先提交或备份这些改动，避免被新的复制内容覆盖。

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
website/               官网、Node.js 预约演示 API 和 Nginx/Compose 部署配置
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

前端单元测试：

```bash
pnpm --dir frontend test
```

官网本地开发、测试和构建使用 Node.js 24+：

```bash
pnpm --dir website install
pnpm --dir website dev
pnpm --dir website test
pnpm --dir website run build
```

官网 SMTP、本地开发和 Nginx/Compose 部署说明见 [`website/README.md`](website/README.md)。

测试必须使用临时目录，不能读写真实的 `~/.hermes-dock`。文件与配置测试优先调用真实解析和持久化逻辑；Docker、网络、Wails runtime 和系统命令只在边界处替换。`cmd/hermes-profile-runner` 的测试应覆盖 `.env` 安全解析、日志前缀与脱敏、重启限制和状态写入。CI 会执行 Go 测试、前端测试、前端构建，并校验生成的 Wails bindings 是否与 Go 方法保持同步。

## MVP 范围

当前包含：

- Docker 和 Docker Compose 检测。
- 首次启动从内置干净模板初始化。
- 标准 compose 生成和高级 override 入口。
- 启动、停止、重启、重建、状态和日志。
- 镜像、端口、控制台认证和资源限制编辑。
- 主模型和 auxiliary 模型配置。
- 百炼按量计费、百炼 Coding Plan、百炼 Token Plan 团队版、智谱按量计费、智谱 Coding Plan、OpenCode Go、DeepSeek 和 Agnes AI 供应商预设。
- 个人微信扫码登录。
- 企业微信 AI Bot WebSocket 配置。
- 飞书 / Lark WebSocket 配置。
- 通道查看、默认通道设置和测试消息发送。
- UI 输出脱敏。
- 写入前本地备份。
- 整实例 `.hdbackup` 导出和覆盖导入，导入前自动备份当前实例。

当前不做：

- 不安装 Docker。
- 不做系统服务安装。
- 不做多实例管理。
- 不做单 profile 多账号平台管理；多个平台身份通过单容器多 profile 隔离运行。
- 不内置真实运行态、日志、会话、缓存、数据库或用户凭据。
- 不做完整 Hermes 平台配置器，只覆盖 MVP 指定平台。
- 不做内置聊天客户端，聊天仍使用 Hermes 控制台。
- 不在普通导航中提供环境变量编辑器；`.env` 默认由结构化配置和平台绑定流程维护，高级编辑可打开。
