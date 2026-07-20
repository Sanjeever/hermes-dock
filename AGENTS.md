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
- 多 profile 目标仍是单 Hermes Docker 容器，不做多 Compose project 或多 Hermes 实例；默认启用的 Dufs 仅作为共享文件 Web 管理 sidecar。
- 只要求用户已安装 Docker，不负责安装 Docker。
- 启动器接管标准 `~/.hermes-dock/docker-compose.yaml`。
- 高级 Docker 自定义放在 `~/.hermes-dock/docker-compose.override.yaml`。
- `~/.hermes-dock/data` 是用户数据，默认永不覆盖。
- 不自动迁移或修补已有 profile 的 `config.yaml`；内置模板只用于创建缺失文件。
- 只做显式保存、绑定或实例迁移，不做静默重置。
- 不把真实运行态、日志、会话、缓存、数据库、auth 文件或微信账号凭据放进内置模板。
- 多 profile 第一版不为每个 profile 单独暴露 HTTP/API/Dashboard 端口；profile gateway 只服务各自平台入口。

## 重要目录

```text
templates/seed-data/       内置干净模板，首次启动释放到 data/
frontend/src/App.tsx       React 主界面
frontend/src/App.css       React 样式
website/                   独立官网、Node.js 预约演示 API 和 Nginx/Compose 部署配置
app.go                     Wails 状态聚合
compose.go                 Compose 生成和容器生命周期
config.go                  config.yaml、模型配置和模型列表
env.go                     .env 读写、合并和脱敏
weixin.go                  个人微信扫码登录
dingtalk_login.go          钉钉扫码绑定
platforms.go               企业微信、飞书、钉钉和通道相关操作
paths.go                   实例路径和 safePath 限制
backup.go                  写入前备份
web.go                     内置 Web 管理、登录会话、RPC 和 WebSocket
hostbridge.go              宿主机命令执行、Token 认证、超时和输出限制
```

多 profile 相关路径约定：

```text
~/.hermes-dock/data/                     default profile 的 Hermes home
~/.hermes-dock/data/profiles/<id>/       非 default profile 的 Hermes home
~/.hermes-dock/launcher/profiles.json    Dock profile registry，事实来源
~/.hermes-dock/launcher/profile-content/<id>.json 内置人格和技能同步基线
~/.hermes-dock/launcher/apply-status.json 应用配置后台任务状态
~/.hermes-dock/launcher/web-server.json  Web 管理配置
~/.hermes-dock/launcher/web-sessions.json Web 登录会话
~/.hermes-dock/launcher/logs/web-server.log Web 访问日志
~/.hermes-dock/launcher/host-bridge.token Host Bridge 随机认证 Token
~/.hermes-dock/launcher/dufs/config.yaml Dufs 配置和共享账号密码哈希
~/.hermes-dock/data/.dock/               runner 派生运行态
~/.hermes-dock/shared/                  默认共享文件目录
```

项目不保留单独的 `docs/` 目录。架构和 MVP 边界必须维护在 `README.md` 和本文件中，避免多份文档漂移。

## 数据安全规则

- 修改任何会影响 `~/.hermes-dock/data` 的逻辑前，先确认不会覆盖用户已有文件。
- `releaseSeedData` 只能创建缺失文件，不能覆盖已有文件。
- 写入 `config.yaml`、`.env`、`docker-compose.yaml` 或高级编辑文件前，应保留备份。
- 写入 profile 的 `config.yaml`、`.env`、`SOUL.md`、`skills/` 或 `launcher/profiles.json` 前，应保留备份。
- 删除非 default profile 时，如果容器正在运行，必须先 `docker compose stop`，再整体打包备份 profile 目录；停止或备份失败则中止删除，删除成功后由用户通过“应用配置”重新启动。
- 整实例 `.hdbackup` 导出用于设备迁移，包含 `.env` 密钥、平台账号凭据、Web 管理配置和 Dufs 账号密码哈希；UI 必须明确提示备份文件包含敏感信息。
- 整实例导出如果容器正在运行，应先 `docker compose stop`，导出结束后 `docker compose start` 恢复，避免备份写入中的用户数据；不要用 `down + up` 做导出恢复，避免意外应用未重建配置。
- 整实例导入是覆盖导入，必须先校验备份、执行 `docker compose down`，再生成当前设备的 pre-import `.hdbackup`；任一步失败都中止导入。
- 整实例备份不包含 `shared`、`launcher/backups`、`launcher/logs`、`launcher/web-sessions.json`、`launcher/apply-status.json`、`launcher/host-bridge.token` 或 `data/.dock` 派生运行态；导入后为当前设备重新生成 Host Bridge Token。
- 不要把密钥写入 `launcher/state.json`。
- Host Bridge Token 只保存在 `launcher/host-bridge.token`，权限为 `0600`，不得返回 UI、写入日志、profile registry 或 runtime manifest。
- 不要把密钥写入 `launcher/profiles.json`、`data/.dock/profiles-runtime.json` 或 `data/.dock/profile-status.json`。
- UI 日志、事件、错误信息中不要输出完整 token、API key、secret。
- Dufs 明文密码不得写入 `launcher/state.json`、Compose、日志或 UI 状态；`launcher/dufs/config.yaml` 只保存 SHA-512 crypt 哈希。
- Web 管理与桌面端保持同等管理能力，会返回和编辑当前 profile 的完整环境配置；访问密码是远程管理边界。
- Web 高级编辑与桌面端一致，开放当前 profile 的 `config.yaml`、`.env` 和全局 `docker-compose.override.yaml`；保存 Compose 覆盖文件需要输入“确认”。
- Web 管理提供与桌面端一致的“恢复出厂设置”危险操作。
- 不要为了兼容失败而吞掉错误；应返回清晰错误，让 UI 展示。
- 高级页“恢复出厂设置”是唯一允许删除 `~/.hermes-dock` 实例数据的流程；必须先 `docker compose down`，失败则中止，不加 `--volumes`，并保留默认 `shared` 目录及其中的用户文件。

## Compose 约定

`docker-compose.yaml` 由启动器生成和维护，当前模板包含：

- Hermes 镜像。
- 单 profile 版本为 `command: gateway run`；多 profile 版本应启动 Hermes Dock runner，由 runner 并行启动多个 profile gateway worker。
- 控制台和网关端口。
- 控制台认证环境变量，控制台固定启用。
- 中国大陆友好的 pip、uv、npm 镜像源。
- 数据目录权限依赖 Hermes 镜像启动脚本定向处理，不要增加每次启动都对整个 `/opt/data` 执行 `chown -R` 的 init service。
- 飞书运行依赖通过 `launcher/helpers/install-feishu-deps` 挂载到 `/etc/cont-init.d/018-install-feishu-deps`，由 s6-overlay 在 profile runner 前执行；uv 下载缓存使用 `data/.dock/uv-cache`。
- 钉钉运行依赖通过 `launcher/helpers/install-dingtalk-deps` 挂载到 `/etc/cont-init.d/020-install-dingtalk-deps`，由 s6-overlay 在 profile runner 前执行；uv 下载缓存使用 `data/.dock/uv-cache`。
- Home Channel 首次对话提示通过 `launcher/helpers/patch-home-channel-prompt` 挂载到 `/etc/cont-init.d/019-patch-home-channel-prompt`，只关闭主动提示，不改变 `/sethome` 或实际投递校验。
- 宿主机控制 helper 通过 `launcher/helpers/hostctl` 挂载到 `/usr/local/bin/hostctl`，容器使用 `host.docker.internal:9877` 访问桌面主进程内的 Host Bridge。
- Host Bridge 默认开启、静默执行且不逐次审批；使用随机 Token 认证，以当前宿主机用户身份运行，不自动提权，Dock 退出时停止。
- Host Bridge 结构化 API 覆盖文件读写/查看/移动、系统通知、文本剪贴板、进程和端口查询、多显示器 PNG 截图、默认应用打开、应用启动，以及轻量桌面自动化；Hermes 应优先使用这些 API，`shell`/`exec` 只作为通用补充。
- 轻量桌面自动化只提供窗口枚举/激活、鼠标和键盘原子动作，不做 UI 控件树、OCR、录制器、后台工作流或动作批处理。鼠标坐标使用指定显示器截图内的局部像素坐标；点击、拖拽、滚动和键盘输入必须校验预期前台窗口。
- 桌面键鼠由所有 profile 共享，Host Bridge 使用 30 秒短租约串行化不同 profile 的自动化操作。Windows 使用原生窗口和输入 API；macOS 依赖“辅助功能”和“屏幕录制”权限；Linux 只在 X11 且已有 `xdotool` 时适度支持，不自动安装依赖或绕过 Wayland 限制。
- 文件读写最大 16 MiB，剪贴板最大 1 MiB，截图最大 25 MiB；截图、剪贴板和通知受宿主机桌面会话及 macOS/Wayland 系统权限约束，失败时返回明确错误，不伪造结果。
- 单 profile 版本使用 `env_file: ./data/.env`；多 profile runner 版本不能用一个全局 `env_file` 表达 profile 运行态密钥。
- `volumes: ./data:/opt/data`。
- 用户可配置宿主机共享文件目录，默认 `~/.hermes-dock/shared`，统一挂载为 `/opt/data/.dock/shared` 并由所有 profile 读写；共享目录结构由用户自行管理。
- 共享目录位于固定镜像的单根 `HERMES_WRITE_SAFE_ROOT=/opt/data` 内；共享目录路径变化属于 Compose 变化，需要重建容器。
- 默认启用 `sigoden/dufs:v0.46.0` sidecar，将同一共享目录挂载到 `/data`，通过 `0.0.0.0:9878` 提供局域网 HTTP 文件管理；默认账号 `qizhihe`、默认密码 `123456`。
- Dufs 使用单个全目录读写账号，允许上传、删除、搜索和归档，关闭 symlink、hash 和 CORS；Unix 宿主机使用当前 UID/GID 运行，根文件系统只读、丢弃 Linux capabilities，并限制 Docker 日志轮转。
- Dufs 设置变化使用独立运行时指纹；仅修改 Dufs 开关、端口或账号时不得重启 Hermes。启动器打开和迁移时只写配置，不自动执行 Docker。
- 资源配额默认值按 Docker daemon 可用资源计算，不直接读取物理机总资源：内存限制为 `max(floor(Docker MemTotal / GiB) - 2, 1)G`，CPU 限制为 Docker `NCPU` 全量并格式化为一位小数，例如 `8.0`。
- 资源配额读取 Docker 失败时使用固定 fallback `4G` / `2.0`；只在首次初始化或配置字段缺失时填充，不覆盖用户已保存值，旧用户已有 `4G` / `2.0` 也保持不变。
- 设置页“使用推荐值”是显式重算入口，按当前 Docker 可用资源填入内存和 CPU，仍需用户保存；`shm_size` 继续默认 `1g`，不动态计算。

容器操作命令约定：

- 启动：`docker compose up -d`
- 停止：`docker compose stop`
- 重启：`docker compose restart hermes`
- 重建：`docker compose up -d --force-recreate --remove-orphans hermes`

“应用配置”必须比较标准 Compose 与 override 的当前指纹和 `LastAppliedComposeHash`。仅 profile `.env`、`config.yaml`、`SOUL.md`、skills、平台绑定或启停状态变化时，复用现有容器并执行 `docker compose restart hermes`；镜像、端口、资源、代理、标准 Compose 或 override 变化时执行重建。容器不存在或状态未知时也执行重建。

“应用配置”使用绑定 runtime manifest generation 的后台任务。Docker 操作完成后持续等待当前 generation 的所有 runnable profile 进入 `running`；超过两分钟只标记启动较慢，不直接失败。任务状态写入 `launcher/apply-status.json` 并通过 `apply:status` 事件推送，桌面端和 Web 端同时使用 `GetAppState` 轮询兜底。任务成功后才清除 `NeedsRebuild`，同一时间只允许一个应用任务。

多 profile 版本中，`SOUL.md`、`skills/`、`config.yaml`、`.env` 和平台绑定保存后不承诺热更新，统一通过“应用配置”重启容器内运行态；保存动作本身不自动应用。

## 架构约定

- Go 后端执行 Docker、文件、备份、平台绑定和模型列表拉取。
- React 前端只保留表单状态和展示状态，保存动作走 Wails Go 方法。
- Wails 事件用于流式输出 Docker 日志、命令进度和微信扫码状态。
- 内置 Web 管理服务运行在 Wails 桌面主进程内，随主进程启动，不做独立 server 二进制或 CLI/headless 服务。静默自动升级是唯一允许的系统定时任务集成，默认关闭，只在用户主动开启后注册，关闭时删除。
- Web 管理默认开启，监听 `0.0.0.0:9876`，默认访问密码 `123456`；只使用访问密码，不使用用户名。
- 关闭窗口会退出桌面主进程并停止 Web 管理，不保持托盘后台常驻。
- Web 业务调用使用白名单 RPC：`POST /api/rpc`，事件使用 `/ws/events`，不要做任意 Go 方法反射调用。
- Web 版 `GetAppState` 不返回完整 `Environment`。
- Web 访问日志只记录启动/停止、登录结果、RPC 方法名和失败摘要，不记录请求体、token、API key、secret。
- 内置模板来自 `templates/seed-data/`，只能包含干净初始文件和 Hermes 内置 skills 快照。
- `launcher/state.json` 只保存启动器元数据和 UI 策略，不保存密钥。
- `launcher/profiles.json` 保存 profile registry，不保存密钥，数组顺序就是 UI 显示顺序。
- `data/.dock/profiles-runtime.json` 是可再生成 runner manifest，不备份。
- `data/.dock/profile-status.json` 是 runner 写入的运行态状态，不备份。
- `data/.dock/` 由宿主机 Dock 和容器 runner 共同写入，目录保持 sticky writable；除容器内的 `/opt/data/.dock/shared` 是外部共享文件挂载点外，其中只能保存可再生成且不含密钥的派生运行态。
- `launcher/backups/` 保存写入前备份。
- `launcher/helpers/` 保存临时 helper，例如微信扫码登录脚本。
- `website/` 的 React 构建产物由 Nginx 提供，Node.js 只提供 `POST /api/demo-requests`；预约通知通过外部 SMTP 发送，不依赖桌面主进程。
- 官网 SMTP 使用 `SMTP_HOST`、`SMTP_PORT`、`SMTP_SECURE`、`SMTP_USER`、`SMTP_PASS`、`SMTP_FROM` 和 `MAIL_TO` 配置；生产值直接写入服务器上的 `/opt/qizhih-website-server/docker-compose.yaml`，不要把真实密钥提交到 Git。
- 官网 `SMTP_SECURE=true` 表示连接即 TLS，`false` 表示必须升级 STARTTLS，不允许降级到明文。
- 官网发件人和收件人只从 Node.js 服务环境读取，不接受前端请求指定；SMTP 完成并明确接受邮件后才向页面返回成功。
- `.github/workflows/deploy-website.yml` 在 `main` 分支的 `website/**` 变化时自动测试、构建并部署官网，也支持 `workflow_dispatch` 手动触发；并发部署必须通过 `website-production` concurrency group 串行化，不在运行中取消部署。
- 官网自动部署使用 `WEBSITE_DEPLOY_HOST`、`WEBSITE_DEPLOY_PORT`、`WEBSITE_DEPLOY_USER`、`WEBSITE_DEPLOY_SSH_KEY` 和 `WEBSITE_DEPLOY_KNOWN_HOSTS` 五个 Actions Secrets，不要在 workflow、仓库文件或日志中写入真实值。
- 官网自动部署上传 `dist/` 和由 esbuild 生成的自包含 Node API bundle，不得上传或覆盖服务器上的 `/opt/qizhih-website-server/docker-compose.yaml`。
- 官网生产 API 目录只保留 `/opt/qizhih-website-server/docker-compose.yaml` 和 `/opt/qizhih-website-server/index.js`；`package.json`、`pnpm-lock.yaml`、模块化源码、测试和 `node_modules` 只用于仓库本地开发与 CI，不上传到生产服务器。
- 官网采用覆盖式前端部署和单文件 API 部署：替换 `/home/nginx/html/qizhih-website`，停止 API 容器，原子替换 `/opt/qizhih-website-server/index.js` 后重新启动。该流程有短暂停机且不自动回滚，失败时应通过 Actions 日志和手工部署流程恢复。

## 多 Profile 设计约定

目标模型：

- 一个 Hermes Compose service 和一个默认启用的 Dufs sidecar service。
- 一个 Hermes 容器；Dufs 不承载 profile 或 Hermes 运行态。
- 一个 Hermes Dock runner 作为容器主进程。
- 多个 Hermes profile gateway 子进程并行运行。
- 每个子进程绑定一个 Hermes profile，服务该 profile 的个人微信、企业微信、飞书 / Lark 或钉钉入口。

profile 目录和 ID：

- `default` profile 使用 `data/` 根目录，默认注册并启用，但允许停用，不允许删除。
- 非 default profile 使用 `data/profiles/<id>/`。
- profile ID 只能使用路径安全 ASCII slug：`a-z`、`0-9`、`-`，创建后不可改；中文只作为显示名。
- 删除非 default profile 后允许重建同名 ID，但如果残留目录存在，不自动复用或覆盖。

profile 隔离：

- 按 profile 隔离：`SOUL.md`、`skills/`、`config.yaml`、`.env`、供应商、模型、平台绑定、通道目录、记忆和会话。
- 全局共享：Docker 镜像、容器名、端口、CPU、内存、shm、共享文件目录、`docker-compose.override.yaml`。
- 模型供应商和 API Key 默认 profile 级隔离。UI 可以提供显式复制模型配置到其他 profile，默认不复制 API Key。
- 平台策略如 `WECOM_DM_POLICY`、`WECOM_GROUP_POLICY`、`WEIXIN_DM_POLICY`、`FEISHU_GROUP_POLICY`、`DINGTALK_REQUIRE_MENTION` 也按 profile 写入各自 `.env`。

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
- 平台绑定不完整、enabled profile 平台身份冲突、`config.yaml` 无法解析等严重错误应在“应用配置”前阻止。
- 同一个 `WECOM_BOT_ID`、`WEIXIN_ACCOUNT_ID`、`FEISHU_APP_ID` 或 `DINGTALK_CLIENT_ID` 不能被多个 enabled profile 同时使用。
- 一个 profile 可以同时绑定个人微信、企业微信、飞书 / Lark 和钉钉。
- runner 给每行子进程日志加 profile 前缀，例如 `[sales] ...`；runner 自身日志使用 `[runner]`。
- runner 状态必须携带当前 runtime manifest 的 `generation`；Dock 不接受其他 generation 的旧状态。
- runner 对异常退出的 profile 做有限自动重启，连续失败后标记 failed，其他 profile 继续运行。
- 无可运行 profile 时 runner 仍保持容器 running。
- 第一版停止、启动、重启仍作用于整个容器，不做单 profile 启停/重启。

UI 和功能边界：

- 首页应以 Profiles 总览为主，同时展示容器运行环境状态。
- profile 优先，平台绑定在 profile 详情内完成。
- `enabled` 只表示参与运行，不影响编辑。
- 保存配置不自动应用运行态；显示未应用变更，由用户手动“应用配置”。
- Profiles 总览支持从一个 profile 向多个 profile 一次性复制模型、人格和指定 skills；API Key 默认不复制。
- 启动器内置 `SOUL.md` 和 skills 支持批量同步：同步内置人格时先备份再直接重置 `SOUL.md`；内置 skills 新增缺失文件、更新未被用户修改的文件，并保留用户修改、自定义 skills 和模板已移除的旧 skills。
- 模型测试和平台测试消息只针对当前 profile。
- 桌面和 Web 高级编辑都以当前 profile 为上下文，可打开当前 profile 的 `config.yaml`、`.env` 和全局 `docker-compose.override.yaml`。
- 第一版不做 Kanban/跨 profile 协作 UI，但目录、ID 和 runner 启动方式必须保持 Hermes 原生 profile/Kanban 兼容。
- 第一版不做按消息内容跨 profile 路由，不做 profile 导入/导出，不做批量创建向导。
- skills 管理、Skill Hub 安装、打开本机技能目录和同步启动器内置最新技能在桌面端和 Web 端都可用。同步内置技能会用启动器模板覆盖当前 profile 的内置技能文件，不删除自定义技能或模板外文件。

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
- 百炼按量计费、百炼 Coding Plan、百炼 Token Plan 团队版、智谱按量计费、智谱 Coding Plan、OpenCode Go、DeepSeek 和 Agnes AI 供应商预设及模型列表拉取。
- 个人微信扫码登录。
- 企业微信 AI Bot WebSocket 配置。
- 飞书 / Lark WebSocket 配置。
- 钉钉 Stream 模式扫码和 AppKey / AppSecret 配置。
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

- `dashscope-payg`：百炼按量计费，`provider: custom`，默认模型 `qwen3.7-max`，模型列表 `https://dashscope.aliyuncs.com/compatible-mode/v1/models`。
- `bailian-coding-plan`：百炼 Coding Plan，`provider: custom`，默认模型 `qwen3.7-max`，模型列表 `https://coding.dashscope.aliyuncs.com/v1/models`。
- `bailian-token-plan-team`：百炼 Token Plan 团队版，`provider: custom`，OpenAI 兼容地址 `https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`，模型名手动填写。
- `zhipu-payg`：智谱按量计费，`provider: custom`，默认模型 `glm-5.2`，模型列表 `https://open.bigmodel.cn/api/paas/v4/models`。
- `zhipu-coding-plan`：智谱 Coding Plan，`provider: custom`，默认模型 `glm-5.2`，模型列表 `https://open.bigmodel.cn/api/coding/paas/v4/models`。
- `opencode-go`：OpenCode Go，`provider: custom`，默认模型 `deepseek-v4-flash`，模型列表 `https://opencode.ai/zen/go/v1/models`。
- `deepseek`：DeepSeek，`provider: deepseek`，默认模型 `deepseek-v4-flash`，模型列表 `https://api.deepseek.com/models`。
- `agnes`：Agnes AI，`provider: custom`，默认模型 `agnes-2.0-flash`，模型列表 `https://apihub.agnes-ai.com/v1/models`。

供应商页负责新增、编辑、禁用供应商和填写 API Key。内置供应商不可删除，自定义供应商被主模型或辅助模型引用时不可删除。模型页只选择供应商 ID 和模型名；主供应商没有 API Key 时允许保存模型选择，但禁止测试。保存供应商或模型配置时，只把当前主模型和辅助模型实际引用的供应商密钥同步到当前 profile `.env`，空密钥不同步，也不清理旧 `.env` 遗留键。

Auxiliary 模型策略由 UI 控制，状态记录在 `launcher/state.json` 的 `ModelAuxiliaryMode`。`auto` 策略下辅助模型保持 `provider: auto` 和空兼容字段；`follow-main` 和 `custom` 才根据引用供应商展开兼容字段。

## 平台绑定

当前稳定版本每个平台只支持一个实例；多 profile 第一版会改为每个 profile 可绑定一个实例，enabled profiles 中平台身份必须唯一：

- 一个 Weixin / WeChat Personal。
- 一个 WeCom AI Bot。
- 一个 Feishu / Lark App。
- 一个 DingTalk App。

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
- 私聊和群聊策略只支持 `open` 和 `closed`；保存企业微信配置时清空旧版本名单字段。
- 多 profile 版本中 `WECOM_BOT_ID` 在 enabled profiles 中必须唯一。

飞书 / Lark：

- 只支持 WebSocket 模式。默认通过扫码自动创建并绑定机器人；根据扫码账号自动识别 `FEISHU_DOMAIN=feishu` 或 `lark`。已有应用在“使用已有应用（高级）”中手动填写 App ID 和 App Secret，不做 webhook。
- 固定写入 `FEISHU_CONNECTION_MODE=websocket`。
- 默认 `FEISHU_ALLOW_ALL_USERS=true`。
- 默认 `FEISHU_GROUP_POLICY=open`。
- 群聊策略只支持 `open` 和 `disabled`，界面显示为“开放”和“关闭”；保存飞书配置时清空旧版本名单字段。
- 整个 Dock 同时只允许一个微信、飞书 / Lark 或钉钉扫码绑定会话。重新扫码只在成功后替换当前 profile 的凭据，旧飞书应用不自动删除。
- 多 profile 版本中 `FEISHU_APP_ID` 在 enabled profiles 中必须唯一。
- 飞书 Python 依赖由 `/etc/cont-init.d/018-install-feishu-deps` 自动安装固定版本，不读取或输出 App Secret。

钉钉：

- 只支持 Stream 模式。支持扫码创建并绑定机器人，或在“使用已有应用（高级）”中手动填写 AppKey 和 AppSecret；不做 webhook。
- 默认 `DINGTALK_ALLOW_ALL_USERS=true`，默认 `DINGTALK_REQUIRE_MENTION=true`，`DINGTALK_ALLOWED_USERS` 保持为空。
- 扫码成功后才替换当前 profile 凭据，AppSecret 不返回 UI 或日志。
- 多 profile 版本中 `DINGTALK_CLIENT_ID` 在 enabled profiles 中必须唯一。
- 钉钉 Python 依赖由 `/etc/cont-init.d/020-install-dingtalk-deps` 自动安装 `dingtalk-stream==0.24.3`、`alibabacloud-dingtalk==2.2.42` 和 `qrcode==7.4.2`，不读取或输出 AppSecret。

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
pnpm --dir website install
pnpm --dir website dev
pnpm --dir website test
pnpm --dir website run build
```

用户明确要求验证时再运行测试或构建。文档类修改通常不需要主动跑测试。

## 发布版本

- 私有源码仓库固定为 [`sqyl2026/hermes-dock`](https://github.com/sqyl2026/hermes-dock)，公开二进制发布仓库固定为 [`sqyl2026/hermes-dock-releases`](https://github.com/sqyl2026/hermes-dock-releases)。
- 应用的更新检查、发布页和安装包下载只使用公开发布仓库，不得依赖私有源码仓库的 Release。
- 检测到新版本后，“立即更新”会下载平台安装包和 `SHA256SUMS.txt`，校验后由独立 `hermes-dock-updater` 在主程序退出后安装并重启；不得停止 Hermes Docker 容器。
- 静默自动升级默认关闭。用户开启后才注册 Windows 计划任务、Linux systemd timer 或 macOS LaunchDaemon，关闭时必须删除；手动“立即更新”不改变该开关。
- 发布产物必须包含对应平台的 `hermes-dock-updater`，Windows 便携版和 Linux tar 包也必须包含 updater。
- 版本标签推送到私有源码仓库后，由 `.github/workflows/build-release.yml` 构建多平台产物，并使用 Actions Secret `RELEASE_REPO_TOKEN` 发布到公开发布仓库。该 Token 只应具有 `sqyl2026/hermes-dock-releases` 的 `Contents: read and write` 权限。
- workflow 自带的 `GITHUB_TOKEN` 对源码仓库保持 `contents: read`，不要为跨仓库发布扩大其权限。
- 发布前先确认工作树干净，并用 `rg` 检查以下三个版本值一致：`app.go` 的 `appVersion`、`frontend/package.json` 的 `version`、`wails.json` 的 `info.productVersion`。
- 版本递增时必须同步更新这三个位置；不得只更新前端或 Wails 产品版本。
- 版本提交使用 `chore(release): bump version to <version>`。提交后创建注释标签 `v<version>`，标签说明为 `Release v<version>`，并推送 `main` 和该标签。
- 推送后确认私有源码仓库的远端 `main` 与 `v<version>` 标签均指向该发布提交，并确认公开发布仓库已创建同版本 Release、完整上传安装包和 `SHA256SUMS.txt`。

## 代码风格

- Go 代码改动后运行 `gofmt`。
- Windows 桌面主进程执行后台外部命令时使用 `backgroundCommand` / `backgroundCommandContext`，避免 Docker、PowerShell 等控制台窗口闪现。
- 前端依赖使用 `pnpm`。
- Python 临时代码如必须使用，遵循用户级约定：用 `uv run python`，不要直接用 `python` 或 `pip`。
- 优先保持改动小而直接，避免过度抽象。
- 不做与任务无关的格式化。
- 不使用破坏性 git 命令。

## 常见坑

- profile `.env` 由 runner 在 profile 启动时读取，可以通过“应用配置”重启现有容器进入运行态；Compose `environment` 变化仍必须重建容器。
- Hermes CLI 可能能从 `/opt/data/.env` 读到配置，但 gateway 运行态依赖进程环境变量。
- `docker compose config` 能看到 env 并不代表当前旧容器已经拿到 env。
- 多 profile 版本不能继续依赖全局 Compose `env_file`，否则多个 profile 的同名平台密钥会互相覆盖。
- 不要通过覆盖 Compose `command` 或 `entrypoint` 安装 Python 依赖；飞书和钉钉依赖分别由 `/etc/cont-init.d/018-install-feishu-deps`、`/etc/cont-init.d/020-install-dingtalk-deps` 完成。
- 飞书依赖 helper 来源是 `launcher/helpers/install-feishu-deps`，脚本修改后必须同步更新 helper 释放和 Compose 迁移相关测试。
- 钉钉依赖 helper 来源是 `launcher/helpers/install-dingtalk-deps`，脚本修改后必须同步更新 helper 释放和 Compose 迁移相关测试。
- Home Channel 提示补丁来源是 `launcher/helpers/patch-home-channel-prompt`，脚本修改后必须同步更新 helper 释放和 Compose 迁移相关测试。
- 非 default profile 如果 `terminal.cwd` 或 `SOUL.md` 仍指向 `/opt/data`，会污染 default profile 根目录。
- Weixin iLink bot 是否能收到普通微信群消息，受 iLink 侧能力限制。
- 现有 `~/.hermes-dock/data/.env` 可能包含旧版本遗留键，默认保留，不要清理用户文件。
