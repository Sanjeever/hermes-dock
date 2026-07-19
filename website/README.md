# 企智盒官网

`website` 包含两部分：

- React + Vite 前端，构建后的 `dist/` 直接上传到现有 Nginx 静态目录。
- 原生 JavaScript Node.js API，源码和生产依赖在构建时合并为单个 `index.js` 发布，使用官方 `node:24-alpine` 镜像运行并通过 SMTP 发送预约通知。

前端包含两个静态页面：

- `/`：官网首页。
- `/manual/`：操作手册，正文维护在 `content/manual.md`。

`content/manual.md` 顶部的 `title`、`description` 和 `updated` 同时驱动页面可见信息与预渲染后的 SEO 元数据。`title` 以“企智盒”开头，`updated` 使用 `YYYY-MM-DD` 格式；二级标题会自动生成页内目录，标题文字应保持唯一。

预约接口为 `POST /api/demo-requests`。项目使用 Node.js 24 或更高版本，依赖使用 pnpm 管理。

## SMTP 配置

生产环境变量直接写在 `docker-compose.yaml` 的 `environment` 中：

```yaml
environment:
  NODE_ENV: production
  PORT: "3000"
  SMTP_HOST: "smtp.example.com"
  SMTP_PORT: "465"
  SMTP_SECURE: "true"
  SMTP_USER: "sender@example.com"
  SMTP_PASS: "replace-with-smtp-password-or-auth-code"
  SMTP_FROM: "sender@example.com"
  MAIL_TO: "recipient@example.com"
```

- `SMTP_SECURE=true` 表示连接时直接使用 TLS，通常对应 465 端口。
- `SMTP_SECURE=false` 表示连接后必须成功升级 STARTTLS，通常对应 587 端口。
- `SMTP_PASS` 应填写 SMTP 密码或邮箱服务商提供的客户端授权码。163 邮箱需要填写客户端授权码，而不是邮箱登录密码。
- `SMTP_FROM` 通常应与 `SMTP_USER` 一致。

仓库中的 Compose 只包含示例值。真实 SMTP 密码只写入服务器上的 `/opt/qizhih-website-server/docker-compose.yaml`，不要把修改后的生产配置提交到 Git。

## 本地开发

安装依赖：

```bash
pnpm install
```

在当前终端设置 SMTP 环境变量：

```bash
export SMTP_HOST=smtp.example.com
export SMTP_PORT=465
export SMTP_SECURE=true
export SMTP_USER=sender@example.com
export SMTP_PASS=replace-with-smtp-password-or-auth-code
export SMTP_FROM=sender@example.com
export MAIL_TO=recipient@example.com
```

同时启动 Node API 和 Vite：

```bash
pnpm dev
```

页面地址：

```text
http://localhost:5173
```

Vite 会把 `/api` 和 `/healthz` 代理到 `127.0.0.1:3000`。本地提交预约会向 `MAIL_TO` 真实发送邮件。

单独启动 API：

```bash
pnpm start
```

构建官网和 Node API：

```bash
pnpm build
```

构建会生成浏览器端和预渲染 bundle，把 React 页面分别写入 `website/dist/index.html` 和 `website/dist/manual/index.html`，并将 Node API 源码及其生产依赖合并为 CommonJS 格式的 `website/dist-server/index.js`。`website/dist/` 是可直接部署到 Nginx 的纯静态产物，`dist-server/index.js` 是部署到生产服务器 `/opt/qizhih-website-server/index.js` 的自包含 API 文件；生产目录没有 `package.json`，Node 会按 CommonJS 加载它。模块化源码、测试、`package.json` 和 `pnpm-lock.yaml` 只保留在仓库及 CI 中。

## 测试

```bash
pnpm test
```

自动测试使用注入的邮件发送器，不会连接真实 SMTP。

## 生产目录

服务器目录约定：

```text
/home/nginx/html/qizhih-website/   前端 dist 内容

/opt/qizhih-website-server/
├── docker-compose.yaml
└── index.js        自动部署替换的自包含 bundle
```

生产服务器只保留 Compose 配置和构建后的 `index.js`。Node 服务不构建业务镜像，`docker-compose.yaml` 使用 `node:24-alpine`，只读挂载 `index.js` 并执行：

```text
node /app/index.js
```

容器启动时不执行 pnpm，也不需要联网安装依赖。

## 服务器首次准备

创建目录和 Docker 网络：

```bash
mkdir -p /home/nginx/html/qizhih-website
mkdir -p /opt/qizhih-website-server
docker network create qizhih-website
```

首次构建并上传 Node API：

```bash
pnpm --dir website build

scp website/dist-server/index.js \
  root@42.194.190.65:/opt/qizhih-website-server/index.js

scp website/docker-compose.yaml \
  root@42.194.190.65:/opt/qizhih-website-server/
```

然后在服务器编辑生产 SMTP 配置：

```bash
vi /opt/qizhih-website-server/docker-compose.yaml
```

启动 Node API：

```bash
cd /opt/qizhih-website-server
docker compose up -d
docker compose ps
docker compose logs --tail=100 server
```

## 现有服务器一次性迁移

自动部署只会替换根目录的 `index.js`，不会修改包含真实 SMTP 配置的 `docker-compose.yaml`。因此，现有服务器仍使用 `server/index.js` 和 pnpm 启动时，必须先手工迁移，再推送这次 workflow 变更。

先在本地构建并上传新的自包含文件：

```bash
pnpm --dir website build

scp website/dist-server/index.js \
  root@42.194.190.65:/opt/qizhih-website-server/index.js.next
```

在服务器备份生产 Compose，然后编辑原文件。保留真实 SMTP 环境变量，只把启动命令和挂载改为：

```yaml
command: ["node", "/app/index.js"]
volumes:
  - ./index.js:/app/index.js:ro
```

同时删除 `NPM_CONFIG_REGISTRY`、旧的 `package.json`、`pnpm-lock.yaml`、`server/` 和 `node-modules` 挂载，以及顶层 `node-modules` volume 声明。执行迁移：

```bash
install -m 600 \
  /opt/qizhih-website-server/docker-compose.yaml \
  /root/qizhih-website-server-compose-backup.yaml

vi /opt/qizhih-website-server/docker-compose.yaml

cd /opt/qizhih-website-server
docker compose config >/dev/null
docker compose down
mv index.js.next index.js
docker compose up -d
docker compose ps
docker compose exec -T server \
  node -e "fetch('http://127.0.0.1:3000/healthz').then((response) => { if (!response.ok) process.exit(1) }).catch(() => process.exit(1))"
```

确认健康检查和预约邮件都正常后，再删除旧文件：

```bash
rm -rf /opt/qizhih-website-server/server \
  /opt/qizhih-website-server/node-modules
rm -f /opt/qizhih-website-server/package.json \
  /opt/qizhih-website-server/pnpm-lock.yaml
```

最终运行 `ls -la /opt/qizhih-website-server`，业务目录中应只剩 `docker-compose.yaml` 和 `index.js`。workflow 会在替换前端或停止 API 前检查这个目录和 Compose；未完成迁移时会直接终止部署。

## 接入现有 Nginx Compose

现有 `/home/nginx/docker-compose.yaml` 中，让 Nginx 加入同一个外部网络：

```yaml
services:
  nginx:
    # 保留现有 image、ports、volumes 等配置
    networks:
      - qizhih-website

networks:
  qizhih-website:
    external: true
```

修改 Compose 后重新创建 Nginx 容器：

```bash
cd /home/nginx
docker compose up -d --force-recreate
```

## Nginx 配置

`ai.sqyl.online` 的 HTTPS `server` 直接提供前端文件，仅把 API 和健康检查反向代理到 Node 容器：

```nginx
server {
    listen 443 ssl;
    server_name ai.sqyl.online;

    ssl_certificate     /etc/nginx/certs/ai.sqyl.online_bundle.crt;
    ssl_certificate_key /etc/nginx/certs/ai.sqyl.online.key;

    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    root /usr/share/nginx/html/qizhih-website;
    index index.html;

    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/css application/javascript application/json image/svg+xml;

    location /api/ {
        proxy_pass http://qizhih-website-server:3000;
        proxy_http_version 1.1;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_connect_timeout 10s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }

    location = /healthz {
        proxy_pass http://qizhih-website-server:3000/healthz;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /assets/ {
        try_files $uri =404;
        expires 1y;
        add_header Cache-Control "public, max-age=31536000, immutable";
    }

    location = / {
        try_files /index.html =404;
        add_header Cache-Control "no-cache";
    }

    location = /index.html {
        try_files $uri =404;
        add_header Cache-Control "no-cache";
    }

    location = /manual {
        return 301 /manual/;
    }

    location = /manual/ {
        try_files /manual/index.html =404;
        add_header Cache-Control "no-cache";
    }

    location = /manual/index.html {
        try_files $uri =404;
        add_header Cache-Control "no-cache";
    }

    location / {
        try_files $uri =404;
    }
}
```

官网包含首页和 `/manual/` 操作手册两个预渲染静态页面，不使用 SPA 路由回退。手册目录页由 `/manual/index.html` 提供，首页和手册 HTML 均不缓存；带哈希的 `/assets/` 继续使用长期缓存。不存在的路径应返回真实 `404`，避免搜索引擎把它识别成 soft 404。`robots.txt` 和 `sitemap.xml` 由前端 `public/` 目录随构建产物发布。

80 端口继续跳转 HTTPS：

```nginx
server {
    listen 80;
    server_name ai.sqyl.online;
    return 301 https://ai.sqyl.online$request_uri;
}
```

应用配置：

```bash
docker exec nginx nginx -t
docker exec nginx nginx -s reload
```

## GitHub Actions 自动部署

`.github/workflows/deploy-website.yml` 在 `main` 分支的 `website/**` 发生变化时自动执行，也支持通过 `workflow_dispatch` 手动触发。workflow 会依次执行：

1. 安装 `website` 依赖。
2. 运行官网测试和构建。
3. 将 `dist/` 打包，并把 Node API 构建为单个 `index.js`，上传到服务器 `/tmp`。
4. 删除并重新创建 `/home/nginx/html/qizhih-website`，然后解压新的前端文件。
5. 在 `/opt/qizhih-website-server` 执行 `docker compose down`。
6. 原子替换 `/opt/qizhih-website-server/index.js`。
7. 执行 `docker compose up -d`，在容器内检查 API，并验证公网首页和 `/healthz`。

自动部署需要在私有源码仓库配置以下 Actions Secrets：

```text
WEBSITE_DEPLOY_HOST
WEBSITE_DEPLOY_PORT
WEBSITE_DEPLOY_USER
WEBSITE_DEPLOY_SSH_KEY
WEBSITE_DEPLOY_KNOWN_HOSTS
```

部署使用覆盖式更新，没有自动回滚，前端和 API 都会短暂停机。workflow 只替换服务器上的 `index.js`，不会覆盖包含真实 SMTP 配置的 `docker-compose.yaml`。生产依赖已经包含在 bundle 中，不需要在服务器另行安装或同步。部署失败时应查看 Actions 日志，并按下方手工流程恢复。

## 手工部署更新

在本地构建并上传前端：

```bash
pnpm --dir website install --frozen-lockfile
pnpm --dir website build

rsync -az --delete website/dist/ \
  root@42.194.190.65:/home/nginx/html/qizhih-website/
```

上传 Node API 单文件 bundle：

```bash
scp website/dist-server/index.js \
  root@42.194.190.65:/opt/qizhih-website-server/index.js.next

ssh root@42.194.190.65 <<'REMOTE'
set -euo pipefail
cd /opt/qizhih-website-server
docker compose down
mv index.js.next index.js
docker compose up -d
docker compose logs --tail=100 server
REMOTE
```

更新时不要覆盖服务器上的 `docker-compose.yaml`，否则会覆盖其中的真实 SMTP 配置。生产依赖变化时只需重新构建并发布 `index.js`；只有 Compose 结构发生变化时才手工合并对应修改。

检查容器内 API：

```bash
docker exec qizhih-website-server \
  node -e "fetch('http://127.0.0.1:3000/healthz').then((response) => response.text()).then(console.log)"
```

检查公网：

```bash
curl -I https://ai.sqyl.online/
curl -I https://ai.sqyl.online/manual/
curl https://ai.sqyl.online/healthz
```

健康检查应返回：

```json
{"ok":true}
```

Node 服务信任一层 Nginx 反向代理，并使用 Nginx 传入的客户端 IP 对预约接口执行每个 IP 每分钟 5 次的限流。Node 容器只通过 Docker 网络暴露 3000 端口，不应把该端口发布到公网。
