#!/bin/sh
set -eu

BASE_PYTHON=/opt/hermes/.venv/bin/python
OCR_VENV=/opt/data/.dock/image-text-ocr-venv
OCR_PYTHON="$OCR_VENV/bin/python"
DEPS=/opt/hermes-dock/runtime-deps
INSTALL_MARKER="$OCR_VENV/.hermes-dock-runtime-deps-sha256"
EXPECTED_MARKER=$(sha256sum "$DEPS/SHA256SUMS" | awk '{print $1}')

echo "[paddleocr-deps] checking dependencies..."

if [ -f "$INSTALL_MARKER" ] && \
    [ "$(cat "$INSTALL_MARKER")" = "$EXPECTED_MARKER" ] && \
    "$OCR_PYTHON" -c 'from importlib.metadata import version; assert version("paddleocr") == "3.7.0"; assert version("paddlepaddle") == "3.1.1"; assert version("paddlex") == "3.7.2"; import paddle; from paddleocr import PaddleOCR' >/dev/null 2>&1; then
    echo "[paddleocr-deps] dependencies already installed"
    exit 0
fi

echo "[paddleocr-deps] installing PaddleOCR runtime..."

uv venv --clear --python "$BASE_PYTHON" "$OCR_VENV"
uv pip install \
    --offline \
    --no-index \
    --no-deps \
    --find-links "$DEPS/wheels" \
    --python "$OCR_PYTHON" \
    --requirements "$DEPS/ocr.lock"

"$OCR_PYTHON" -c 'import paddle; from paddleocr import PaddleOCR'
printf '%s\n' "$EXPECTED_MARKER" > "$INSTALL_MARKER.tmp"
mv "$INSTALL_MARKER.tmp" "$INSTALL_MARKER"

echo "[paddleocr-deps] installation completed"
