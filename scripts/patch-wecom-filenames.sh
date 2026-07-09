#!/bin/sh
set -eu

PYTHON=/opt/hermes/.venv/bin/python
TARGET=/opt/hermes/gateway/platforms/wecom.py

echo "[wecom-filename-patch] checking adapter..."

if [ ! -f "$TARGET" ]; then
    echo "[wecom-filename-patch] missing $TARGET" >&2
    exit 1
fi

if grep -q "MAX_WECOM_CACHE_BASENAME_BYTES" "$TARGET"; then
    echo "[wecom-filename-patch] already applied"
    exit 0
fi

"$PYTHON" - <<'PY'
from pathlib import Path

target = Path("/opt/hermes/gateway/platforms/wecom.py")
text = target.read_text()

if "MAX_WECOM_CACHE_BASENAME_BYTES" in text:
    raise SystemExit(0)

if "import hashlib\n" not in text:
    if "import base64\n" not in text:
        raise RuntimeError("WeCom adapter layout changed: import marker not found")
    text = text.replace("import base64\n", "import base64\nimport hashlib\n", 1)

backoff_needle = "RECONNECT_BACKOFF = [2, 5, 10, 30, 60]\n"
backoff_patch = backoff_needle + '''
# ext4 limits each path component to 255 bytes. cache_document_from_bytes()
# prepends "doc_<uuid12>_", so WeCom filenames must leave room for that prefix.
WECOM_CACHE_PREFIX_BYTES = len("doc_") + 12 + len("_")
MAX_WECOM_CACHE_BASENAME_BYTES = 255 - WECOM_CACHE_PREFIX_BYTES
'''
if backoff_needle not in text:
    raise RuntimeError("WeCom adapter layout changed: reconnect backoff marker not found")
text = text.replace(backoff_needle, backoff_patch, 1)

decode_needle = '''    @staticmethod
    def _decode_base64(data: str) -> bytes:
'''
helper_patch = '''    @staticmethod
    def _truncate_utf8(value: str, max_bytes: int) -> str:
        raw = value.encode("utf-8")[:max_bytes]
        return raw.decode("utf-8", "ignore").rstrip()

    @classmethod
    def _sanitize_inbound_filename(cls, filename: str, fallback: str = "document") -> str:
        name = Path(unquote(str(filename or ""))).name
        name = name.replace("\\x00", "").strip()
        if not name or name in {".", ".."}:
            name = fallback

        raw = name.encode("utf-8")
        if len(raw) <= MAX_WECOM_CACHE_BASENAME_BYTES:
            return name

        suffix = Path(name).suffix
        digest = hashlib.sha1(raw).hexdigest()[:8]
        marker = f"_{digest}"
        budget = MAX_WECOM_CACHE_BASENAME_BYTES - len(suffix.encode("utf-8")) - len(marker.encode("ascii"))
        if budget <= 0:
            return cls._truncate_utf8(name, MAX_WECOM_CACHE_BASENAME_BYTES)

        stem = name[:-len(suffix)] if suffix else name
        return f"{cls._truncate_utf8(stem, budget)}{marker}{suffix}"

'''
if decode_needle not in text:
    raise RuntimeError("WeCom adapter layout changed: decode marker not found")
text = text.replace(decode_needle, helper_patch + decode_needle, 1)

base64_needle = '''            filename = str(media.get("filename") or media.get("name") or "wecom_file")
            return cache_document_from_bytes(raw, filename), mimetypes.guess_type(filename)[0] or "application/octet-stream"
'''
base64_patch = '''            filename = self._sanitize_inbound_filename(media.get("filename") or media.get("name"), "wecom_file")
            return cache_document_from_bytes(raw, filename), mimetypes.guess_type(filename)[0] or "application/octet-stream"
'''
if base64_needle not in text:
    raise RuntimeError("WeCom adapter layout changed: base64 filename marker not found")
text = text.replace(base64_needle, base64_patch, 1)

download_needle = '''        filename = self._guess_filename(url, headers.get("content-disposition"), content_type)
        return cache_document_from_bytes(raw, filename), content_type
'''
download_patch = '''        filename = self._sanitize_inbound_filename(self._guess_filename(url, headers.get("content-disposition"), content_type))
        return cache_document_from_bytes(raw, filename), content_type
'''
if download_needle not in text:
    raise RuntimeError("WeCom adapter layout changed: downloaded filename marker not found")
text = text.replace(download_needle, download_patch, 1)

target.write_text(text)
PY

echo "[wecom-filename-patch] applied"
