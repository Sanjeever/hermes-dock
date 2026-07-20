#!/bin/sh
set -eu

PYTHON=/opt/hermes/.venv/bin/python
UV_CACHE_DIR=/opt/data/.dock/uv-cache
export UV_CACHE_DIR

echo "[dingtalk-deps] checking dependencies..."

if "$PYTHON" -c 'from importlib.metadata import version; assert version("dingtalk-stream") == "0.24.3"; assert version("alibabacloud-dingtalk") == "2.2.42"; assert version("qrcode") == "7.4.2"' >/dev/null 2>&1; then
    echo "[dingtalk-deps] dependencies already installed"
    exit 0
fi

echo "[dingtalk-deps] installing dingtalk-stream, alibabacloud-dingtalk and qrcode..."

uv pip install \
    --python "$PYTHON" \
    'dingtalk-stream==0.24.3' \
    'alibabacloud-dingtalk==2.2.42' \
    'qrcode==7.4.2'

"$PYTHON" -c 'import dingtalk_stream, qrcode'

echo "[dingtalk-deps] installation completed"
