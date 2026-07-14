#!/bin/sh
set -eu

PYTHON=/opt/hermes/.venv/bin/python
TARGET=/opt/hermes/gateway/run.py

echo "[home-channel-prompt-patch] checking gateway..."

if [ ! -f "$TARGET" ]; then
    echo "[home-channel-prompt-patch] missing $TARGET" >&2
    exit 1
fi

if grep -q "HERMES_DOCK_SUPPRESS_HOME_CHANNEL_PROMPT" "$TARGET"; then
    echo "[home-channel-prompt-patch] already applied"
    exit 0
fi

"$PYTHON" - <<'PY'
from pathlib import Path

target = Path("/opt/hermes/gateway/run.py")
text = target.read_text()

if "HERMES_DOCK_SUPPRESS_HOME_CHANNEL_PROMPT" in text:
    raise SystemExit(0)

needle = '''        # One-time prompt if no home channel is set for this platform
        # Skip for webhooks - they deliver directly to configured targets (github_comment, etc.)
        if not history and source.platform and source.platform != Platform.LOCAL and source.platform != Platform.WEBHOOK:
'''
replacement = '''        # One-time prompt if no home channel is set for this platform
        # Skip for webhooks - they deliver directly to configured targets (github_comment, etc.)
        suppress_home_channel_prompt = os.getenv(
            "HERMES_DOCK_SUPPRESS_HOME_CHANNEL_PROMPT", ""
        ).lower() in {"1", "true", "yes"}
        if not suppress_home_channel_prompt and not history and source.platform and source.platform != Platform.LOCAL and source.platform != Platform.WEBHOOK:
'''
if needle not in text:
    raise RuntimeError("Hermes gateway layout changed: home channel prompt marker not found")

target.write_text(text.replace(needle, replacement, 1))
PY

"$PYTHON" -m py_compile "$TARGET"

echo "[home-channel-prompt-patch] applied"
