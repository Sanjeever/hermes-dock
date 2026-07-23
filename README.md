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
- 可视化管理资源配额、共享文件、Web 管理和局域网文件管理。
- 通过模型、部署和平台绑定表单写入必要配置，不向普通用户提供环境变量编辑页。
- 可视化配置主模型和 auxiliary 模型。
- 支持百炼按量计费、百炼 Coding Plan、百炼 Token Plan 团队版、智谱按量计费、智谱 Coding Plan、火山方舟 Coding Plan、火山方舟 Agent Plan、OpenCode Go、DeepSeek 和 Agnes AI 模型供应商预设。
- 支持通过 API Key 拉取模型列表并选择模型。
- 支持个人微信 Weixin / WeChat Personal 扫码登录。
- 支持企业微信 AI Bot WebSocket 配置。
- 支持飞书 / Lark WebSocket 配置。
- 支持钉钉 Stream 模式，可扫码创建机器人或手动填写 AppKey / AppSecret。
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
- Hermes 原生 Dashboard 默认关闭，Hermes 服务不向宿主机发布 8642/9119 端口；Hermes 与 Dufs 使用不同的 Compose 网络。
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
    runtime-deps/
      cp313-v2/
    helpers/
      hermes-profile-runner
      verify-runtime-deps
      install-feishu-deps
      install-dingtalk-deps
      patch-home-channel-prompt
      patch-wecom-filenames
```

`data/` 是用户数据，也是 `default` profile 的 Hermes home。非 default profile 使用 `data/profiles/<id>/`，保持和 Hermes 原生 profile 结构兼容。Hermes Dock 不会自动迁移或修补已有 profile 的 `config.yaml`；内置模板只用于创建缺失文件。除用户明确保存配置或执行整实例覆盖导入外，已有配置不会被写入。

`launcher/` 是启动器自己的元数据目录。这里保存状态、profile registry、备份和临时 helper，不应该放用户业务数据或密钥。

`shared/` 是默认共享文件目录，在容器内固定挂载为 `/opt/data/.dock/shared`，由所有 profile 共同读写。用户可以在基础设置中改为其他宿主机绝对路径，目录结构和文件内容由用户自行管理。

Dufs 默认开启并将同一目录挂载为 `/data`，通过局域网 HTTP 端口 `9878` 提供文件管理。默认账号为 `qizhihe`、默认密码为 `123456`，建议首次使用时立即修改；密码只以 SHA-512 crypt 哈希保存在 `launcher/dufs/config.yaml`，不会写入启动器状态或 Compose。Dufs 适合可信局域网，不应直接暴露到公网；需要原生文件系统挂载时可另行使用操作系统自带 SMB 服务。

Web 管理配置保存在 `launcher/web-server.json`，登录会话保存在 `launcher/web-sessions.json`，访问日志保存在 `launcher/logs/web-server.log`。首次创建时 Web 管理默认开启，监听 `0.0.0.0:9876`，默认访问密码为 `123456`；用户可在设置页修改密码。关闭窗口会退出桌面主进程并停止 Web 管理。

启动器会检查公开发布仓库中的稳定版本。发现新版本后可直接点击“立即更新”，启动器会下载对应平台安装包、使用 `SHA256SUMS.txt` 校验、退出并交给独立的 `hermes-dock-updater` 安装，然后重新启动。安装阶段不停止 Hermes Docker 容器；新版本启动后会同步所有助手的内置人格和技能，其中内置技能会用当前模板覆盖内容不同的同名文件，并保留自定义技能。如果同步产生了变化，且 Hermes 在升级前和应用前都确认正在运行，启动器会自动应用配置，期间可能短暂重启或重建 Hermes；已停止或状态未知的服务不会被自动启动。静默自动升级默认关闭，只有用户在“设置 → 软件更新”中主动开启后才会注册系统定时任务，关闭时删除任务。Windows 使用当前管理员用户的最高权限计划任务，Linux 使用 systemd system timer，macOS 使用 LaunchDaemon；更新状态和应用内错误保存在 `launcher/update.json`，独立 updater 和定时更新错误保存在 `launcher/updates/last-error`，详细安装日志保存在 `launcher/logs/update.log`。

`data/.dock/` 保存 runner 的派生运行清单和运行状态。宿主机上的这些文件可由 Hermes Dock 重新生成，不是用户业务数据；容器内的 `/opt/data/.dock/shared` 是绑定到外部共享目录的挂载点。

设置页的数据迁移功能会导出快速 `.hdbackup` 单文件，用于把当前实例迁移到其他设备。备份保留 profile、人格、技能、记忆、会话、任务、平台账号、用户项目和源码，以及 profile registry、Web 管理、Dufs、标准 Compose 和 Compose override 配置，因此也包含 `.env` 密钥、平台账号凭据和远程访问凭据。导出会跳过共享目录、运行日志、Web 登录会话、旧备份、`data/.dock/` 派生运行态，以及可重新生成的缓存、虚拟环境、`node_modules`、临时文件、模型列表缓存和检查点；用户项目中的 `.git` 和交付文件不会被通用排除。导出时如果容器正在运行，会先 `docker compose stop`，导出完成后再 `docker compose start` 恢复原运行状态，避免备份到写入中的文件。导入是实例覆盖流程：先执行 `docker compose down`，再以相同的快速策略自动生成当前设备的导入前备份，最后恢复备份内容并重新生成标准 `docker-compose.yaml`。新版仍可导入旧版包含缓存和依赖目录的 `.hdbackup`。

## 多 Profile 设计

当前多 profile 实现会在一个 Docker 容器内并行运行多个 Hermes profile gateway worker，让不同 profile 绑定不同的个人微信、企业微信 AI Bot、飞书 / Lark 或钉钉应用，并隔离人格、记忆、模型、skills、平台凭据和通道。

运行规则：

- `default` profile 使用 `data/` 根目录，默认进入 profile 列表并参与运行，但允许停用。
- 非 default profile 使用 `data/profiles/<id>/`。
- profile ID 使用路径安全 ASCII slug，例如 `sales`、`support`；中文只作为显示名。
- 每个 enabled profile 如果绑定了完整平台身份，就由 runner 启动对应 gateway。
- enabled 但未绑定平台的 profile 不启动，状态显示为未配置平台。
- 同一个企业微信 Bot、个人微信账号、飞书 App 或钉钉 AppKey 不能被多个 enabled profile 同时使用。
- 一个 profile 可以同时绑定个人微信、企业微信、飞书和钉钉，表示同一个助手服务多个入口。
- 平台入口固定归属一个 profile，第一版不做按消息内容跨 profile 路由。
- 配置保存后只写入文件，不自动应用运行态；用户手动点击“应用配置”后统一生效。
- 可以从一个助手向多个助手批量复制主模型、辅助模型、人格和指定技能；供应商密钥默认不复制。
- 可以把启动器最新版内置人格和技能同步到多个助手；内置人格会先备份再重置，内置技能会先备份并用当前模板覆盖内容不同的同名文件，自定义技能和旧技能会保留。

隔离边界：

- 按 profile 隔离：`SOUL.md`、`skills/`、`config.yaml`、`.env`、供应商、模型、平台绑定、通道目录、记忆和会话。
- 全局共享：Docker 镜像、Dufs 端口、容器名、CPU、内存、shm、`docker-compose.override.yaml`。
- 模型供应商和 API Key 默认按 profile 隔离；UI 可以提供显式“复制模型配置到其他 profile”，默认不复制 API Key。
- 平台策略如 `WECOM_DM_POLICY`、`WECOM_GROUP_POLICY`、`WEIXIN_DM_POLICY`、`FEISHU_GROUP_POLICY`、`DINGTALK_REQUIRE_MENTION` 也按 profile 写入各自 `.env`。

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
- Hermes profile gateway 运行态，但不向宿主机发布原生 HTTP/API/Dashboard 端口。
- Hermes 原生 Dashboard 固定关闭。
- 内存、CPU 和 shm 限制。
- `./data:/opt/data` 数据挂载。
- 可配置的宿主机共享文件目录固定挂载到 `/opt/data/.dock/shared`，默认使用 `~/.hermes-dock/shared`，由所有 profile 共同读写。
- 默认启用固定版本 `sigoden/dufs:v0.46.0`，将同一共享目录挂载到 `/data` 并通过 `0.0.0.0:9878` 提供轻量 Web 文件管理。
- 数据目录权限由 Hermes 镜像启动脚本定向处理，不在每次应用配置时对整个 `data/` 执行递归 `chown`。
- Hermes Dock 按发布平台内置飞书和钉钉所需的 CPython 3.13 Linux wheelhouse：Windows/Linux amd64 携带 `linux/amd64`，macOS arm64 携带 `linux/arm64`。启动器在需要应用配置时释放到 `launcher/runtime-deps/<version>/`，并只读挂载到 `/opt/hermes-dock/runtime-deps`。
- `launcher/helpers/verify-runtime-deps` 挂载到 `/etc/cont-init.d/016-verify-runtime-deps`，在安装前校验 Python 版本、容器架构和所有文件的 SHA-256；校验失败会明确中止，不会联网补包。
- `launcher/helpers/install-feishu-deps` 和 `launcher/helpers/install-dingtalk-deps` 分别在 s6 初始化阶段从本地 wheelhouse 严格离线安装飞书、钉钉运行依赖。
- `image-text-ocr` 技能直接捆绑 PP-OCRv6_small 模型；首次识别时由技能脚本联网下载经过版本和哈希锁定的 PaddleOCR 3.7.0 CPU 依赖，安装到 `data/.dock/image-text-ocr-venv`，后续识别和容器重建复用该环境。图片和模型始终在本地处理，运行时不下载模型。
- `captcha-ocr` 技能使用 ddddocr beta 模型识别字符和基础算术静态图片验证码；首次识别时下载包含模型且经过版本和哈希锁定的依赖，安装到 `data/.dock/captcha-ocr-venv`，后续离线复用。验证码不会发送到外部服务，滑块、点选、拼图及其他反自动化验证不在处理范围内。
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

“应用配置”会比较 Hermes、Dufs 和 override 的当前指纹与上次成功应用的指纹。只修改 profile 的 `.env`、`config.yaml`、`SOUL.md`、skills、平台绑定或启停状态时，复用现有容器并重启 `hermes` 服务；镜像、资源、代理、共享目录或 Compose override 变化时才重新创建 Hermes 容器。只修改 Dufs 开关、端口或共享账号时仅更新 Dufs，不重启 Hermes。应用过程作为后台任务运行并绑定当前 runtime generation；超过两分钟只提示启动较慢，仍会继续等待。当前 generation 的所有 runnable profile 上报 `running` 后才标记成功并自动刷新桌面端和 Web 端状态。

## 模型供应商

供应商配置独立保存在当前 profile 的 `config.yaml` 顶层 `providers` 中，`model.provider` 和辅助模型的 `provider` 字段只引用供应商 ID。启动器保存时会把当前引用供应商的 `base_url`、`api_mode` 和 `api_key` 展开回 `model` / `auxiliary`，兼容 Hermes 当前运行态。

MVP 内置十个供应商实例：

- `dashscope-payg`：百炼按量计费，默认模型 `qwen3.7-max`。
- `bailian-coding-plan`：百炼 Coding Plan，默认模型 `qwen3.7-max`。
- `bailian-token-plan-team`：百炼 Token Plan 团队版，使用 OpenAI 兼容接口，模型名手动填写。
- `zhipu-payg`：智谱按量计费，默认模型 `glm-5.2`。
- `zhipu-coding-plan`：智谱 Coding Plan，默认模型 `glm-5.2`。
- `volcengine-ark-coding-plan`：火山方舟 Coding Plan，默认模型 `doubao-seed-2.0-code`。
- `volcengine-ark-agent-plan`：火山方舟 Agent Plan，默认模型 `doubao-seed-2.0-code`。
- `opencode-go`：OpenCode Go，默认模型 `deepseek-v4-flash`。
- `deepseek`：DeepSeek，默认模型 `deepseek-v4-flash`。
- `agnes`：Agnes AI，默认模型 `agnes-2.0-flash`。

供应商页负责新增、编辑、禁用供应商，以及填写 API Key、接口地址、API 模式和模型列表地址。模型页只选择已配置的供应商和模型名。保存供应商或模型配置时，启动器只把当前主模型和辅助模型实际引用的供应商密钥同步到当前 profile `.env` 的 `DASHSCOPE_API_KEY`、`ZHIPU_API_KEY`、`ARK_API_KEY`、`ARK_AGENT_PLAN_API_KEY`、`OPENCODE_GO_API_KEY`、`DEEPSEEK_API_KEY` 或 `AGNES_API_KEY`，供对应 profile 运行态读取。火山方舟 Coding Plan 与 Agent Plan 是不同订阅，分别使用 `ARK_API_KEY` 和 `ARK_AGENT_PLAN_API_KEY`，密钥不共用。

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

Hermes Dock 会在容器初始化阶段通过包元数据快速检查 `lark-oapi==1.5.3` 和 `qrcode==7.4.2`。缺少时，`/etc/cont-init.d/018-install-feishu-deps` 只从随安装包提供的本地 wheelhouse 安装到 `/opt/hermes/.venv`，然后执行 import 验证；整个过程不访问包索引，也不读取或输出飞书 App Secret。

### 钉钉

MVP 只支持钉钉 Stream 模式，不配置 webhook 回调。可以通过扫码创建并绑定机器人，也可以在“使用已有应用（高级）”中手动填写 AppKey 和 AppSecret。二维码仅用于登录流程，不会将 AppSecret 返回给界面或写入日志。多 profile 版本中，每个 profile 可以绑定一个钉钉应用，enabled profiles 中 `DINGTALK_CLIENT_ID` 必须唯一。默认策略：

- `DINGTALK_ALLOW_ALL_USERS=true`
- `DINGTALK_REQUIRE_MENTION=true`

扫码成功后才会替换当前 profile 的凭据；保存平台配置后需要由用户手动“应用配置”使运行态生效。Hermes Dock 会在容器初始化阶段检查 `dingtalk-stream==0.24.3`、`alibabacloud-dingtalk==2.2.42` 和 `qrcode==7.4.2`。缺少时，`/etc/cont-init.d/020-install-dingtalk-deps` 只从随安装包提供的本地 wheelhouse 安装到 `/opt/hermes/.venv` 并执行 import 验证；该过程不访问包索引，也不读取或输出 AppSecret。

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

本仓库只保留与 Hermes Dock 日常维护直接相关的项目级技能。产品运行时提供给 Hermes 的 skills 位于 `templates/seed-data/skills/`，不受本节影响。

| 技能 | 用途 | 使用边界 |
| --- | --- | --- |
| `systematic-debugging` | 基于证据定位跨 Go、Wails、React、Docker 和 runner 的故障根因 | 遇到 bug、测试失败或异常行为时使用；先复现和验证假设，再实施最小修复。 |
| `requesting-code-review` | 由独立 Agent 对高风险实现做只读审查 | 多 profile、备份迁移、Host Bridge、自动升级和复杂故障修复完成后使用。 |
| `copywriting` | 起草官网、发布页和功能介绍等营销文案 | 仅用于需要转化目标的页面，不用于技术文档或普通界面字段。 |
| `copy-editing` | 审校和刷新已有营销文案 | 保留既有事实，改善官网、发布页和功能页的清晰度、用户收益与行动引导。 |

界面、性能和设计决策以 `AGENTS.md` 中的项目约束及具体任务需求为准，不在项目范围常驻通用设计风格、图像生成或 Next.js 导向的技能，避免普通 React / Wails 改动被不适用的默认流程约束。

更新保留技能：

```bash
npx -y skills update copywriting copy-editing systematic-debugging requesting-code-review --project --yes
```

## 项目结构

```text
app.go                 Wails 应用入口和状态聚合
compose.go             Docker Compose 生成和生命周期操作
config.go              Hermes config.yaml 读写、模型供应商和模型列表
env.go                 data/.env 读写和脱敏
weixin.go              个人微信扫码登录 helper 和凭据保存
platforms.go           企业微信、飞书配置、通道和测试消息
dingtalk_login.go      钉钉扫码绑定、凭据保存和会话控制
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
- 资源限制、共享文件、Web 管理和局域网文件管理配置。
- 主模型和 auxiliary 模型配置。
- 百炼按量计费、百炼 Coding Plan、百炼 Token Plan 团队版、智谱按量计费、智谱 Coding Plan、火山方舟 Coding Plan、火山方舟 Agent Plan、OpenCode Go、DeepSeek 和 Agnes AI 供应商预设。
- 个人微信扫码登录。
- 企业微信 AI Bot WebSocket 配置。
- 飞书 / Lark WebSocket 配置。
- 钉钉 Stream 模式扫码和 AppKey / AppSecret 配置。
- 通道查看、默认通道设置和测试消息发送。
- UI 输出脱敏。
- 写入前本地备份。
- 快速迁移 `.hdbackup` 导出和覆盖导入，导入前自动快速备份当前实例。

当前不做：

- 不安装 Docker。
- 不做系统服务安装。
- 不做多实例管理。
- 不做单 profile 多账号平台管理；多个平台身份通过单容器多 profile 隔离运行。
- 不内置真实运行态、日志、会话、缓存、数据库或用户凭据。
- 不做完整 Hermes 平台配置器，只覆盖 MVP 指定平台。
- 不做内置聊天客户端，消息交互通过已绑定的平台完成。
- 不在普通导航中提供环境变量编辑器；`.env` 默认由结构化配置和平台绑定流程维护，高级编辑可打开。
