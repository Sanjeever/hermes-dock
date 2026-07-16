#!/bin/sh
set -eu

PYTHON=/opt/hermes/.venv/bin/python
UV_CACHE_DIR=/opt/data/.dock/uv-cache
export UV_CACHE_DIR

echo "[feishu-deps] checking dependencies..."

if "$PYTHON" -c 'from importlib.metadata import version; assert version("lark-oapi") == "1.5.3"; assert version("qrcode") == "7.4.2"' >/dev/null 2>&1; then
    echo "[feishu-deps] dependencies already installed"
    exit 0
fi

echo "[feishu-deps] installing lark-oapi and qrcode..."

uv pip install \
    --python "$PYTHON" \
    'lark-oapi==1.5.3' \
    'qrcode==7.4.2'

"$PYTHON" -c 'import lark_oapi, qrcode'

echo "[feishu-deps] installation completed"
