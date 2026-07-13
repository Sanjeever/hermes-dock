---
name: hermes-dock-host
description: "Operate the Windows, macOS, or Linux host running Hermes Dock: inspect host information, run commands, access files, launch applications, and open URLs."
version: 1.0.0
platforms: [linux, macos, windows]
metadata:
  hermes:
    tags: [hermes-dock, host, shell, windows, macos, linux]
    related_skills: [hermes-dock]
---

# 操作企智盒宿主机

你运行在 Docker 容器中。需要操作运行企智盒的 Windows、macOS 或 Linux 宿主机时，必须使用 `hostctl`，不要在容器终端中直接执行目标命令。

宿主机操作已经由用户统一授权，无需逐次请求批准。直接执行所需操作；失败时返回 `hostctl` 的真实错误，不要假装成功。

## 确认宿主机环境

不知道宿主机系统、用户目录或路径格式时，先运行：

```bash
hostctl info
```

不要根据容器的 Linux 环境猜测宿主机系统。

## 执行命令

通过宿主机 Shell 执行命令：

```bash
hostctl shell 'command'
hostctl shell --cwd '/host/project/path' 'command'
hostctl shell --timeout 300 'command'
```

不需要 Shell 语法时，优先使用参数数组执行，减少引号和转义问题：

```bash
hostctl exec --cwd '/host/project/path' git status --short
hostctl exec code '/host/project/path'
```

`hostctl shell` 使用宿主机对应的 Shell：Windows 使用 PowerShell，macOS 使用 zsh，Linux 使用 sh。

## 文件、应用和 URL

文件操作也通过宿主机命令完成。先根据 `hostctl info` 返回的操作系统选择命令：

```bash
# macOS
hostctl shell 'open "https://example.com"'

# Linux
hostctl shell 'xdg-open "https://example.com"'

# Windows PowerShell
hostctl shell 'Start-Process "https://example.com"'
```

宿主机路径和容器路径不同。`/opt/data` 是容器路径；macOS 通常使用 `/Users/...`，Linux 通常使用 `/home/...`，Windows 使用 `C:\Users\...`。Windows 路径包含反斜杠时，优先通过 `hostctl exec` 传递参数。

## 运行约束

- 默认超时为 120 秒，`--timeout` 最大为 1800 秒。
- 命令以启动企智盒的当前用户身份执行，不会自动获得管理员、UAC、sudo 或 root 权限。
- `stdout` 和 `stderr` 分别最多返回 1 MiB，超过部分会被截断。
- 企智盒未运行或“宿主机控制”已关闭时，`hostctl` 会失败。
- 停止或重建当前 Hermes 容器会中断正在进行的会话；此类操作优先引导用户使用企智盒界面。
