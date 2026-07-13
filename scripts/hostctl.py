#!/usr/bin/env python3

import argparse
import json
import os
import sys
import urllib.error
import urllib.request


BASE_URL = "http://host.docker.internal:9877"
TOKEN_PATH = "/opt/hermes-dock/host-bridge.token"


def request(path, payload=None):
    with open(TOKEN_PATH, encoding="utf-8") as token_file:
        token = token_file.read().strip()
    data = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request(
        BASE_URL + path,
        data=data,
        headers={
            "Authorization": "Bearer " + token,
            "Content-Type": "application/json",
        },
        method="GET" if payload is None else "POST",
    )
    try:
        with urllib.request.urlopen(req) as response:
            return json.load(response)
    except urllib.error.HTTPError as error:
        try:
            detail = json.load(error).get("error", str(error))
        except (ValueError, AttributeError):
            detail = str(error)
        raise RuntimeError(detail) from error
    except urllib.error.URLError as error:
        raise RuntimeError("无法连接 Hermes Dock 宿主机控制服务") from error


def run_command(payload):
    result = request("/v1/exec", payload)
    if result.get("stdout"):
        sys.stdout.write(result["stdout"])
    if result.get("stderr"):
        sys.stderr.write(result["stderr"])
    if result.get("timed_out"):
        print("宿主机命令执行超时", file=sys.stderr)
    return result.get("exit_code", 1)


def main():
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

    args = parser.parse_args()
    try:
        if args.action == "info":
            print(json.dumps(request("/v1/info"), ensure_ascii=False, indent=2))
            return 0
        if args.action == "shell":
            return run_command({
                "command": args.command,
                "cwd": args.cwd,
                "timeout_seconds": args.timeout,
            })
        return run_command({
            "program": args.program,
            "args": args.args,
            "cwd": args.cwd,
            "timeout_seconds": args.timeout,
        })
    except (OSError, RuntimeError) as error:
        print(str(error), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
