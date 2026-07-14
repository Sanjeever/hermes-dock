# 企智盒官网

`website` 是 React + Vite 单页站点，静态页面和预约接口一起部署到 Cloudflare Workers。

本项目使用 Node.js 24 或更高版本。

## 本地开发

复制 SMTP 配置模板并填写真实值：

```bash
cp .dev.vars.example .dev.vars
```

`SMTP_SECURE=true` 表示连接时直接使用 TLS，通常对应 465 端口；`SMTP_SECURE=false` 表示连接后必须成功升级 STARTTLS，通常对应 587 端口。Cloudflare Workers 不允许连接 SMTP 25 端口。

```bash
pnpm install
pnpm dev
```

本地提交预约会向 `MAIL_TO` 真实发送邮件。自动测试不会连接 SMTP：

```bash
pnpm test
pnpm build
```

## 部署配置

GitHub Actions 需要以下 Repository Secrets：

```text
CLOUDFLARE_ACCOUNT_ID
CLOUDFLARE_API_TOKEN
```

首次部署后，在 Cloudflare Worker 的 Variables and Secrets 中配置：

```text
SMTP_HOST
SMTP_PORT
SMTP_SECURE
SMTP_USER
SMTP_PASS
SMTP_FROM
MAIL_TO
```

`SMTP_PASS` 必须使用 Secret；邮箱地址如不希望出现在控制台明文中，也应使用 Secret。推送到 `main` 且 `website/**` 发生变化时，GitHub Actions 会自动测试、构建并部署 `qizhih-box-website`。
