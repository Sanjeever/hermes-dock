#!/bin/sh
set -eu

BASE_PYTHON=/opt/hermes/.venv/bin/python
OCR_VENV=/opt/data/.dock/image-text-ocr-venv
OCR_PYTHON="$OCR_VENV/bin/python"
UV_CACHE_DIR=/opt/data/.dock/uv-cache
export UV_CACHE_DIR

echo "[paddleocr-deps] checking dependencies..."

if "$OCR_PYTHON" -c 'from importlib.metadata import version; assert version("paddleocr") == "3.7.0"; assert version("paddlepaddle") == "3.1.1"; assert version("paddlex") == "3.7.2"; import paddle; from paddleocr import PaddleOCR' >/dev/null 2>&1; then
    echo "[paddleocr-deps] dependencies already installed"
    exit 0
fi

python_tag=$("$BASE_PYTHON" -c 'import sys; print(f"cp{sys.version_info.major}{sys.version_info.minor}")')
architecture=$(uname -m)
case "$python_tag/$architecture" in
    cp311/x86_64|cp311/aarch64|cp313/x86_64|cp313/aarch64)
        ;;
    *)
        echo "[paddleocr-deps] unsupported runtime: $python_tag/$architecture" >&2
        exit 1
        ;;
esac

paddle_wheel="https://paddle-whl.bj.bcebos.com/stable/cpu/paddlepaddle/paddlepaddle-3.1.1-${python_tag}-${python_tag}-linux_${architecture}.whl"

echo "[paddleocr-deps] installing PaddleOCR runtime..."

uv venv --clear --python "$BASE_PYTHON" "$OCR_VENV"
uv pip install \
    --python "$OCR_PYTHON" \
    "$paddle_wheel" \
    'paddleocr==3.7.0' \
    'paddlex==3.7.2'

"$OCR_PYTHON" -c 'import paddle; from paddleocr import PaddleOCR'

echo "[paddleocr-deps] installation completed"
