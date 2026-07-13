#!/usr/bin/env python3

import argparse
import base64
import json
import os
import sys
import tempfile
import urllib.error
import urllib.parse
import urllib.request


BASE_URL = "http://host.docker.internal:9877"
TOKEN_PATH = "/opt/hermes-dock/host-bridge.token"
MAX_FILE_BYTES = 16 * 1024 * 1024


def raw_request(path, payload=None, method=None):
    with open(TOKEN_PATH, encoding="utf-8") as token_file:
        token = token_file.read().strip()
    data = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request(
        BASE_URL + path,
        data=data,
        headers={
            "Authorization": "Bearer " + token,
            "Content-Type": "application/json",
            "X-Hermes-Profile": os.environ.get("HERMES_DOCK_PROFILE", "default"),
        },
        method=method or ("GET" if payload is None else "POST"),
    )
    try:
        with urllib.request.urlopen(req) as response:
            return response.read(), response.headers.get("Content-Type", ""), dict(response.headers.items())
    except urllib.error.HTTPError as error:
        try:
            detail = json.loads(error.read()).get("error", str(error))
        except (ValueError, AttributeError):
            detail = str(error)
        raise RuntimeError(detail) from error
    except urllib.error.URLError as error:
        raise RuntimeError("无法连接 Hermes Dock 宿主机控制服务") from error


def request_json(path, payload=None, method=None):
    data, _, _ = raw_request(path, payload, method)
    return json.loads(data)


def request_bytes(path, payload):
    data, content_type, headers = raw_request(path, payload)
    if content_type.split(";", 1)[0] != "image/png":
        raise RuntimeError("宿主机没有返回 PNG 截图")
    return data, headers


def print_json(value):
    print(json.dumps(value, ensure_ascii=False, indent=2))


def write_container_file(path, data):
    parent = os.path.dirname(os.path.abspath(path))
    os.makedirs(parent, exist_ok=True)
    temporary = None
    try:
        with tempfile.NamedTemporaryFile(prefix=".hostctl-", dir=parent, delete=False) as output:
            temporary = output.name
            output.write(data)
        os.replace(temporary, path)
    finally:
        if temporary and os.path.exists(temporary):
            os.remove(temporary)


def read_file_input(path):
    if path:
        with open(path, "rb") as source:
            data = source.read(MAX_FILE_BYTES + 1)
    else:
        data = sys.stdin.buffer.read(MAX_FILE_BYTES + 1)
    if len(data) > MAX_FILE_BYTES:
        raise RuntimeError("文件超过 16 MiB 限制")
    return data


def run_command(payload):
    result = request_json("/v1/exec", payload)
    if result.get("stdout"):
        sys.stdout.write(result["stdout"])
    if result.get("stderr"):
        sys.stderr.write(result["stderr"])
    if result.get("timed_out"):
        print("宿主机命令执行超时", file=sys.stderr)
    return result.get("exit_code", 1)


def add_file_commands(subparsers):
    file_parser = subparsers.add_parser("file", help="读写宿主机文件")
    actions = file_parser.add_subparsers(dest="file_action", required=True)

    read_parser = actions.add_parser("read")
    read_parser.add_argument("path")
    read_parser.add_argument("--output")

    write_parser = actions.add_parser("write")
    write_parser.add_argument("path")
    write_parser.add_argument("--input")
    write_parser.add_argument("--no-create-parents", action="store_true")
    write_parser.add_argument("--no-overwrite", action="store_true")

    stat_parser = actions.add_parser("stat")
    stat_parser.add_argument("path")

    list_parser = actions.add_parser("list")
    list_parser.add_argument("path")

    mkdir_parser = actions.add_parser("mkdir")
    mkdir_parser.add_argument("path")

    move_parser = actions.add_parser("move")
    move_parser.add_argument("source")
    move_parser.add_argument("target")
    move_parser.add_argument("--create-parents", action="store_true")
    move_parser.add_argument("--overwrite", action="store_true")


def add_desktop_commands(subparsers):
    notify_parser = subparsers.add_parser("notify", help="发送宿主机通知")
    notify_parser.add_argument("message", nargs="?")
    notify_parser.add_argument("--title", default="Hermes")
    notify_parser.add_argument("--stdin", action="store_true")

    clipboard_parser = subparsers.add_parser("clipboard", help="操作宿主机文本剪贴板")
    clipboard_actions = clipboard_parser.add_subparsers(dest="clipboard_action", required=True)
    clipboard_actions.add_parser("get")
    clipboard_set = clipboard_actions.add_parser("set")
    clipboard_set.add_argument("text", nargs="?")
    clipboard_set.add_argument("--stdin", action="store_true")

    subparsers.add_parser("displays", help="列出宿主机显示器")
    screenshot_parser = subparsers.add_parser("screenshot", help="截取宿主机屏幕")
    screenshot_parser.add_argument("--display", type=int, default=0)
    screenshot_parser.add_argument("--output")
    screenshot_parser.add_argument("--json", action="store_true", dest="json_output")

    open_parser = subparsers.add_parser("open", help="使用宿主机默认应用打开目标")
    open_parser.add_argument("target")

    launch_parser = subparsers.add_parser("launch", help="启动宿主机应用")
    launch_parser.add_argument("--cwd", default="")
    launch_parser.add_argument("program")
    launch_parser.add_argument("args", nargs=argparse.REMAINDER)


def add_rpa_commands(subparsers):
    rpa_parser = subparsers.add_parser("rpa", help="查询和释放桌面自动化能力")
    rpa_actions = rpa_parser.add_subparsers(dest="rpa_action", required=True)
    rpa_actions.add_parser("info")
    rpa_actions.add_parser("release")

    window_parser = subparsers.add_parser("window", help="查询和激活宿主机窗口")
    window_actions = window_parser.add_subparsers(dest="window_action", required=True)
    window_actions.add_parser("list")
    window_actions.add_parser("active")
    window_activate = window_actions.add_parser("activate")
    window_activate.add_argument("id")

    mouse_parser = subparsers.add_parser("mouse", help="操作宿主机鼠标")
    mouse_actions = mouse_parser.add_subparsers(dest="mouse_action", required=True)
    mouse_actions.add_parser("position")
    mouse_move = mouse_actions.add_parser("move")
    add_mouse_point(mouse_move)
    mouse_move.add_argument("--duration", type=int, default=0)

    mouse_click = mouse_actions.add_parser("click")
    add_mouse_point(mouse_click)
    mouse_click.add_argument("--button", choices=("left", "right", "middle"), default="left")
    mouse_click.add_argument("--count", type=int, default=1)
    mouse_click.add_argument("--duration", type=int, default=0)
    mouse_click.add_argument("--expect-window", required=True)

    mouse_drag = mouse_actions.add_parser("drag")
    mouse_drag.add_argument("--display", type=int, default=0)
    mouse_drag.add_argument("--from-x", type=int, required=True)
    mouse_drag.add_argument("--from-y", type=int, required=True)
    mouse_drag.add_argument("--to-x", type=int, required=True)
    mouse_drag.add_argument("--to-y", type=int, required=True)
    mouse_drag.add_argument("--button", choices=("left", "right", "middle"), default="left")
    mouse_drag.add_argument("--duration", type=int, default=0)
    mouse_drag.add_argument("--expect-window", required=True)

    mouse_scroll = mouse_actions.add_parser("scroll")
    add_mouse_point(mouse_scroll)
    mouse_scroll.add_argument("--dx", type=int, default=0)
    mouse_scroll.add_argument("--dy", type=int, default=0)
    mouse_scroll.add_argument("--expect-window", required=True)

    keyboard_parser = subparsers.add_parser("keyboard", help="操作宿主机键盘")
    keyboard_actions = keyboard_parser.add_subparsers(dest="keyboard_action", required=True)
    keyboard_press = keyboard_actions.add_parser("press")
    keyboard_press.add_argument("key")
    keyboard_press.add_argument("--count", type=int, default=1)
    keyboard_press.add_argument("--interval", type=int, default=0)
    keyboard_press.add_argument("--expect-window", required=True)

    keyboard_hotkey = keyboard_actions.add_parser("hotkey")
    keyboard_hotkey.add_argument("keys", nargs="+")
    keyboard_hotkey.add_argument("--expect-window", required=True)

    keyboard_type = keyboard_actions.add_parser("type")
    keyboard_type.add_argument("text", nargs="?")
    keyboard_type.add_argument("--stdin", action="store_true")
    keyboard_type.add_argument("--interval", type=int, default=0)
    keyboard_type.add_argument("--expect-window", required=True)


def add_mouse_point(parser):
    parser.add_argument("--display", type=int, default=0)
    parser.add_argument("--x", type=int, required=True)
    parser.add_argument("--y", type=int, required=True)


def add_system_commands(subparsers):
    process_parser = subparsers.add_parser("process", help="查询宿主机进程")
    process_actions = process_parser.add_subparsers(dest="process_action", required=True)
    process_list = process_actions.add_parser("list")
    process_list.add_argument("--name", default="")
    process_get = process_actions.add_parser("get")
    process_get.add_argument("pid", type=int)

    port_parser = subparsers.add_parser("port", help="查询宿主机端口")
    port_actions = port_parser.add_subparsers(dest="port_action", required=True)
    port_list = port_actions.add_parser("list")
    port_list.add_argument("--listening", action="store_true")
    port_get = port_actions.add_parser("get")
    port_get.add_argument("port", type=int)


def build_parser():
    parser = argparse.ArgumentParser(description="操作 Hermes Dock 所在宿主机")
    subparsers = parser.add_subparsers(dest="action", required=True)
    subparsers.add_parser("info", help="显示宿主机信息")

    shell_parser = subparsers.add_parser("shell", help="通过宿主机 Shell 执行命令")
    shell_parser.add_argument("--cwd", default="")
    shell_parser.add_argument("--timeout", type=int, default=120)
    shell_parser.add_argument("command")

    exec_parser = subparsers.add_parser("exec", help="不经过 Shell 执行宿主机程序")
    exec_parser.add_argument("--cwd", default="")
    exec_parser.add_argument("--timeout", type=int, default=120)
    exec_parser.add_argument("program")
    exec_parser.add_argument("args", nargs=argparse.REMAINDER)

    add_file_commands(subparsers)
    add_desktop_commands(subparsers)
    add_system_commands(subparsers)
    add_rpa_commands(subparsers)
    return parser


def handle_file(args):
    if args.file_action == "read":
        result = request_json("/v1/files/read", {"path": args.path})
        data = base64.b64decode(result["content_base64"])
        if args.output:
            write_container_file(args.output, data)
            print(args.output)
            return 0
        try:
            sys.stdout.write(data.decode("utf-8"))
        except UnicodeDecodeError as error:
            raise RuntimeError("二进制文件请使用 --output 保存到容器") from error
        return 0
    if args.file_action == "write":
        data = read_file_input(args.input)
        result = request_json("/v1/files/write", {
            "path": args.path,
            "content_base64": base64.b64encode(data).decode("ascii"),
            "create_parents": not args.no_create_parents,
            "overwrite": not args.no_overwrite,
        })
        print_json(result)
        return 0
    if args.file_action in ("stat", "list", "mkdir"):
        print_json(request_json(f"/v1/files/{args.file_action}", {"path": args.path}))
        return 0
    print_json(request_json("/v1/files/move", {
        "source": args.source,
        "target": args.target,
        "create_parents": args.create_parents,
        "overwrite": args.overwrite,
    }))
    return 0


def handle_desktop(args):
    if args.action == "notify":
        message = sys.stdin.read() if args.stdin else args.message
        if not message:
            raise RuntimeError("通知内容不能为空")
        print_json(request_json("/v1/notify", {"title": args.title, "message": message}))
        return 0
    if args.action == "clipboard":
        if args.clipboard_action == "get":
            result = request_json("/v1/clipboard/text")
            sys.stdout.write(result["text"])
        else:
            text = sys.stdin.read() if args.stdin or args.text is None else args.text
            print_json(request_json("/v1/clipboard/text", {"text": text}))
        return 0
    if args.action == "displays":
        print_json(request_json("/v1/displays"))
        return 0
    if args.action == "screenshot":
        profile_home = os.environ.get("HERMES_DOCK_PROFILE_HOME", "/opt/data")
        output = args.output or os.path.join(profile_home, "tmp", "host-screen.png")
        data, headers = request_bytes("/v1/screenshot", {"display": args.display})
        write_container_file(output, data)
        if args.json_output:
            print_json({
                "path": output,
                "display": int(headers["X-Screenshot-Display"]),
                "width": int(headers["X-Screenshot-Width"]),
                "height": int(headers["X-Screenshot-Height"]),
            })
        else:
            print(output)
        return 0
    if args.action == "open":
        print_json(request_json("/v1/open", {"target": args.target}))
        return 0
    if args.action == "launch":
        print_json(request_json("/v1/launch", {
            "program": args.program,
            "args": args.args,
            "cwd": args.cwd,
        }))
        return 0
    return None


def handle_system(args):
    if args.action == "process":
        query = {"pid": args.pid} if args.process_action == "get" else {"name": args.name}
        print_json(request_json("/v1/processes?" + urllib.parse.urlencode(query)))
        return 0
    if args.action == "port":
        query = {"port": args.port} if args.port_action == "get" else {"listening": str(args.listening).lower()}
        print_json(request_json("/v1/ports?" + urllib.parse.urlencode(query)))
        return 0
    return None


def handle_rpa(args):
    if args.action == "rpa":
        if args.rpa_action == "info":
            print_json(request_json("/v1/rpa/info"))
        else:
            print_json(request_json("/v1/rpa/release", {}))
        return 0
    if args.action == "window":
        if args.window_action == "list":
            print_json(request_json("/v1/rpa/windows"))
        elif args.window_action == "active":
            print_json(request_json("/v1/rpa/windows/active"))
        else:
            print_json(request_json("/v1/rpa/windows/activate", {"id": args.id}))
        return 0
    if args.action == "mouse":
        if args.mouse_action == "position":
            print_json(request_json("/v1/rpa/mouse"))
            return 0
        payload = {
            "action": args.mouse_action,
            "display": args.display,
            "duration_ms": getattr(args, "duration", 0),
            "expected_window_id": getattr(args, "expect_window", ""),
        }
        if args.mouse_action == "drag":
            payload.update({
                "from_x": args.from_x,
                "from_y": args.from_y,
                "to_x": args.to_x,
                "to_y": args.to_y,
                "button": args.button,
            })
        else:
            payload.update({"x": args.x, "y": args.y})
            if args.mouse_action == "click":
                payload.update({"button": args.button, "count": args.count})
            elif args.mouse_action == "scroll":
                payload.update({"dx": args.dx, "dy": args.dy})
        print_json(request_json("/v1/rpa/mouse", payload))
        return 0
    if args.action == "keyboard":
        payload = {
            "action": args.keyboard_action,
            "expected_window_id": args.expect_window,
        }
        if args.keyboard_action == "press":
            payload.update({"key": args.key, "count": args.count, "interval_ms": args.interval})
        elif args.keyboard_action == "hotkey":
            payload["keys"] = args.keys
        else:
            text = sys.stdin.read() if args.stdin or args.text is None else args.text
            payload.update({"text": text, "interval_ms": args.interval})
        print_json(request_json("/v1/rpa/keyboard", payload))
        return 0
    return None


def main():
    args = build_parser().parse_args()
    try:
        if args.action == "info":
            print_json(request_json("/v1/info"))
            return 0
        if args.action == "shell":
            return run_command({"command": args.command, "cwd": args.cwd, "timeout_seconds": args.timeout})
        if args.action == "exec":
            return run_command({"program": args.program, "args": args.args, "cwd": args.cwd, "timeout_seconds": args.timeout})
        if args.action == "file":
            return handle_file(args)
        result = handle_desktop(args)
        if result is not None:
            return result
        result = handle_system(args)
        if result is not None:
            return result
        result = handle_rpa(args)
        if result is not None:
            return result
        raise RuntimeError("未知操作")
    except (OSError, RuntimeError, ValueError) as error:
        print(str(error), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
