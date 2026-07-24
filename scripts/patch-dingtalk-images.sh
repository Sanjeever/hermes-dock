#!/bin/sh
set -eu

PYTHON=/opt/hermes/.venv/bin/python
TARGET=/opt/hermes/gateway/platforms/dingtalk.py
EXPECTED_SHA256=444915b052ae9c922fcc76708ad73acc8844e928a811b162f98e4a67b9f22d19
PATCH_MARKER=HERMES_DOCK_DINGTALK_IMAGE_PATCH_V1

echo "[dingtalk-image-patch] checking adapter..."

if [ ! -f "$TARGET" ]; then
    echo "[dingtalk-image-patch] missing $TARGET" >&2
    exit 1
fi

if grep -q "$PATCH_MARKER" "$TARGET"; then
    "$PYTHON" -m py_compile "$TARGET"
    echo "[dingtalk-image-patch] already applied"
    exit 0
fi

actual_sha256=$(sha256sum "$TARGET" | awk '{print $1}')
if [ "$actual_sha256" != "$EXPECTED_SHA256" ]; then
    echo "[dingtalk-image-patch] unsupported Hermes DingTalk adapter: $actual_sha256" >&2
    exit 1
fi

"$PYTHON" - <<'PY'
from pathlib import Path

target = Path("/opt/hermes/gateway/platforms/dingtalk.py")
text = target.read_text()

import_needle = """import asyncio
import json
import logging
import os
import re
import traceback
import uuid
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Set
"""
import_replacement = """import asyncio
import json
import logging
import mimetypes
import os
import re
import traceback
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional, Set
"""

method_needle = '''    async def send_image_file(
        self,
        chat_id: str,
        image_path: str,
        caption: Optional[str] = None,
        reply_to: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
        **kwargs,
    ) -> SendResult:
        """DingTalk webhook replies cannot send local image files directly."""
        return SendResult(
            success=False,
            error=(
                "DingTalk session webhook replies do not support local image uploads. "
                "Only markdown/text replies are supported without OpenAPI media upload."
            ),
        )
'''

method_replacement = '''    # HERMES_DOCK_DINGTALK_IMAGE_PATCH_V1
    async def _dingtalk_image_failure(
        self,
        chat_id: str,
        error: str,
        metadata: Optional[Dict[str, Any]],
    ) -> SendResult:
        logger.warning("[%s] Image send failed: %s", self.name, error)
        notice = await self.send(
            chat_id=chat_id,
            content="图片发送失败，请稍后重试或改用共享文件下载。",
            reply_to="dingtalk-image-failure",
            metadata=metadata,
        )
        if not notice.success:
            logger.error("[%s] Failed to deliver image error notice", self.name)
        return SendResult(success=False, error=error)

    def _dingtalk_image_target(self, chat_id: str) -> tuple[str, Dict[str, Any], str]:
        message = self._message_contexts.get(chat_id)
        if not message:
            raise RuntimeError("DingTalk image send requires an incoming message context")

        robot_code = (
            getattr(message, "robot_code", "") or self._robot_code
        )
        if not robot_code:
            raise RuntimeError("Missing robotCode for DingTalk image send")

        conversation_type = str(
            getattr(message, "conversation_type", "1") or "1"
        )
        if conversation_type == "2":
            conversation_id = (
                getattr(message, "conversation_id", "") or chat_id
            )
            if not conversation_id:
                raise RuntimeError("Missing conversation ID for DingTalk group image")
            return (
                "https://api.dingtalk.com/v1.0/robot/groupMessages/send",
                {"openConversationId": conversation_id},
                robot_code,
            )

        sender_staff_id = getattr(message, "sender_staff_id", "") or ""
        if not sender_staff_id:
            raise RuntimeError("Missing sender staff ID for DingTalk direct image")
        return (
            "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend",
            {"userIds": [sender_staff_id]},
            robot_code,
        )

    async def _upload_dingtalk_image(self, image_path: str) -> str:
        token = await self._get_access_token()
        if not token:
            raise RuntimeError("Failed to obtain DingTalk access token")
        if not self._http_client:
            raise RuntimeError("HTTP client not initialized")

        path = Path(image_path)
        content_type = mimetypes.guess_type(path.name)[0] or "application/octet-stream"
        with path.open("rb") as image:
            response = await self._http_client.post(
                "https://oapi.dingtalk.com/media/upload",
                params={"access_token": token, "type": "image"},
                files={"media": (path.name, image, content_type)},
                timeout=60.0,
            )
        if response.status_code >= 300:
            raise RuntimeError(
                f"DingTalk media upload returned HTTP {response.status_code}"
            )
        payload = response.json()
        media_id = str(payload.get("media_id") or "")
        if not media_id:
            raise RuntimeError("DingTalk media upload response missing media_id")
        return media_id

    async def send_image_file(
        self,
        chat_id: str,
        image_path: str,
        caption: Optional[str] = None,
        reply_to: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
        **kwargs,
    ) -> SendResult:
        """Upload and send a local image through DingTalk Robot OpenAPI."""
        try:
            safe_path = self.validate_media_delivery_path(image_path)
            if not safe_path:
                return await self._dingtalk_image_failure(
                    chat_id, "Invalid local image path", metadata
                )

            path = Path(safe_path)
            if path.suffix.lower() not in {".jpg", ".jpeg", ".png", ".gif"}:
                return await self._dingtalk_image_failure(
                    chat_id, "Unsupported DingTalk image type", metadata
                )
            if path.stat().st_size > 20 * 1024 * 1024:
                return await self._dingtalk_image_failure(
                    chat_id, "DingTalk images must not exceed 20 MiB", metadata
                )

            endpoint, target, robot_code = self._dingtalk_image_target(chat_id)
            media_id = await self._upload_dingtalk_image(str(path))
            token = await self._get_access_token()
            if not token:
                raise RuntimeError("Failed to obtain DingTalk access token")
            if not self._http_client:
                raise RuntimeError("HTTP client not initialized")

            response = await self._http_client.post(
                endpoint,
                headers={
                    "x-acs-dingtalk-access-token": token,
                    "Content-Type": "application/json",
                },
                json={
                    "robotCode": robot_code,
                    "msgKey": "sampleImageMsg",
                    # DingTalk accepts a mediaId in sampleImageMsg.photoURL.
                    "msgParam": json.dumps({"photoURL": media_id}),
                    **target,
                },
                timeout=15.0,
            )
            if response.status_code >= 300:
                raise RuntimeError(
                    f"DingTalk image send returned HTTP {response.status_code}"
                )
            payload = response.json()
            if (
                payload.get("success") is False
                or payload.get("errcode") not in (None, 0, "0")
                or str(payload.get("code") or "").lower()
                not in ("", "0", "ok", "success")
            ):
                raise RuntimeError("DingTalk image send rejected the request")
            message_id = str(
                payload.get("processQueryKey") or uuid.uuid4().hex[:12]
            )
            logger.info("[%s] Image sent to DingTalk", self.name)
            return SendResult(success=True, message_id=message_id)
        except RuntimeError as exc:
            error = str(exc)
            return await self._dingtalk_image_failure(chat_id, error, metadata)
        except Exception as exc:
            error = f"DingTalk image send failed ({type(exc).__name__})"
            return await self._dingtalk_image_failure(chat_id, error, metadata)
'''

if text.count(import_needle) != 1:
    raise RuntimeError("Hermes DingTalk import marker not found exactly once")
if text.count(method_needle) != 1:
    raise RuntimeError("Hermes DingTalk send_image_file marker not found exactly once")

text = text.replace(import_needle, import_replacement, 1)
text = text.replace(method_needle, method_replacement, 1)
target.write_text(text)
PY

"$PYTHON" -m py_compile "$TARGET"

echo "[dingtalk-image-patch] applied"
