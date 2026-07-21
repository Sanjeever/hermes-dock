#!/bin/sh
set -eu

DEPS=/opt/hermes-dock/runtime-deps
PYTHON=/opt/hermes/.venv/bin/python

echo "[runtime-deps] verifying bundled dependencies..."

expected_python=$(cat "$DEPS/python-version")
actual_python=$("$PYTHON" -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")')
if [ "$actual_python" != "$expected_python" ]; then
    echo "[runtime-deps] unsupported Python: $actual_python (expected $expected_python)" >&2
    exit 1
fi

expected_arch=$(cat "$DEPS/platform")
actual_arch=$(uname -m)
if [ "$actual_arch" != "$expected_arch" ]; then
    echo "[runtime-deps] unsupported architecture: $actual_arch (expected $expected_arch)" >&2
    exit 1
fi

(cd "$DEPS" && sha256sum -c SHA256SUMS >/dev/null)

echo "[runtime-deps] bundled dependencies verified"
