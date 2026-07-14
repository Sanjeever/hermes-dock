# 企智盒宣传册复刻

这是一个不依赖构建工具的静态 HTML 宣传册。屏幕预览使用 `1024px` 设计基准；PDF 导出固定为单页 A4 竖版。

## 使用

图标使用本地捆绑的 [Lucide](https://lucide.dev/) `0.468.0`。首次使用或在新机器上打开前，先执行：

```bash
pnpm --dir brochure-replica install
```

随后直接在浏览器打开 `index.html` 即可预览。

## 导出 A4 PDF（推荐）

使用 Chromium 自动生成，避免 Firefox 或系统打印设置造成的缩放、边距和分页差异：

```bash
pnpm --dir brochure-replica export:pdf
```

首次导出前，需要安装 Chromium：

```bash
pnpm --dir brochure-replica exec playwright install chromium chromium-headless-shell
```

生成文件位于 `brochure-replica/dist/企智盒宣传册-A4.pdf`。导出器固定启用 A4、零页边距和背景图形。

如需从已生成的 PDF 导出 300 DPI PNG：

```bash
pnpm --dir brochure-replica export:png
```

生成文件位于 `brochure-replica/dist/企智盒宣传册-A4.png`，尺寸约为 `2480 × 3508`（300 DPI）。

## 使用 Firefox 打印

Firefox 适合预览或临时导出。请使用 A4 纵向、100% 缩放、无边距，关闭页眉页脚并开启背景图形。打印样式已固定为一张 A4 页面，但正式对外交付仍建议使用上方的自动导出命令。

## 编辑入口

- 文案和模块结构：`index.html`
- 颜色、尺寸和响应式样式：`styles.css`
- 品牌标志：`logo.svg`
- 通用图标：Lucide 的 `data-lucide` 名称，定义在 `index.html`

机器人、浮动消息卡、笔记本、服务器和二维码均由 CSS 绘制，不依赖原宣传册的整图背景。
