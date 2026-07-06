#!/bin/sh
set -eu

PYTHON=/opt/hermes/.venv/bin/python

echo "[feishu-deps] checking dependencies..."

if "$PYTHON" -c 'import lark_oapi, qrcode' >/dev/null 2>&1; then
    echo "[feishu-deps] dependencies already installed"
    exit 0
fi

echo "[feishu-deps] installing lark-oapi and qrcode..."

uv pip install \
    --python "$PYTHON" \
    --no-cache-dir \
    'lark-oapi==1.5.3' \
    'qrcode==7.4.2'

"$PYTHON" -c 'import lark_oapi, qrcode'

echo "[feishu-deps] installation completed"
