#!/bin/sh
set -eu

PYTHON=/opt/hermes/.venv/bin/python
TARGET=/opt/hermes/gateway/platforms/dingtalk.py
EXPECTED_BASE_SHA256=444915b052ae9c922fcc76708ad73acc8844e928a811b162f98e4a67b9f22d19
EXPECTED_V1_SHA256=842ee304cacec43ea3369c79d0bc10c73b7fdc207dfb5928e96183c0a5f0a04f
EXPECTED_V2_SHA256=7f2dfd4044ef536d68742a39b57b3e4df6923bdbe2830b9afacaf1bc2679263c
PATCH_MARKER=HERMES_DOCK_DINGTALK_MEDIA_PATCH_V2

echo "[dingtalk-media-patch] checking adapter..."

if [ ! -f "$TARGET" ]; then
    echo "[dingtalk-media-patch] missing $TARGET" >&2
    exit 1
fi

actual_sha256=$(sha256sum "$TARGET" | awk '{print $1}')
if [ "$actual_sha256" = "$EXPECTED_V2_SHA256" ]; then
    "$PYTHON" -m py_compile "$TARGET"
    echo "[dingtalk-media-patch] already applied"
    exit 0
fi

if [ "$actual_sha256" != "$EXPECTED_BASE_SHA256" ] && [ "$actual_sha256" != "$EXPECTED_V1_SHA256" ]; then
    echo "[dingtalk-media-patch] unsupported Hermes DingTalk adapter: $actual_sha256" >&2
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

    async def send_document(
        self,
        chat_id: str,
        file_path: str,
        caption: Optional[str] = None,
        file_name: Optional[str] = None,
        reply_to: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
        **kwargs,
    ) -> SendResult:
        """DingTalk webhook replies cannot send local file attachments directly."""
        return SendResult(
            success=False,
            error=(
                "DingTalk session webhook replies do not support local file attachments. "
                "Only markdown/text replies are supported without OpenAPI message send."
            ),
        )
'''

method_replacement = '''    # HERMES_DOCK_DINGTALK_MEDIA_PATCH_V2
    async def _dingtalk_media_failure(
        self,
        chat_id: str,
        error: str,
        metadata: Optional[Dict[str, Any]],
        kind: str,
    ) -> SendResult:
        label = {
            "image": "图片",
            "audio": "音频",
            "video": "视频",
        }.get(kind, "文件")
        logger.warning("[%s] %s send failed: %s", self.name, kind.title(), error)
        notice = await self.send(
            chat_id=chat_id,
            content=f"{label}发送失败，请稍后重试或改用共享文件下载。",
            reply_to=f"dingtalk-{kind}-failure",
            metadata=metadata,
        )
        if not notice.success:
            logger.error("[%s] Failed to deliver %s error notice", self.name, kind)
        return SendResult(success=False, error=error)

    def _dingtalk_media_target(self, chat_id: str) -> tuple[str, Dict[str, Any], str]:
        message = self._message_contexts.get(chat_id)
        if not message:
            raise RuntimeError("DingTalk media send requires an incoming message context")

        robot_code = (
            getattr(message, "robot_code", "") or self._robot_code
        )
        if not robot_code:
            raise RuntimeError("Missing robotCode for DingTalk media send")

        conversation_type = str(
            getattr(message, "conversation_type", "1") or "1"
        )
        if conversation_type == "2":
            conversation_id = (
                getattr(message, "conversation_id", "") or chat_id
            )
            if not conversation_id:
                raise RuntimeError("Missing conversation ID for DingTalk group media")
            return (
                "https://api.dingtalk.com/v1.0/robot/groupMessages/send",
                {"openConversationId": conversation_id},
                robot_code,
            )

        sender_staff_id = getattr(message, "sender_staff_id", "") or ""
        if not sender_staff_id:
            raise RuntimeError("Missing sender staff ID for DingTalk direct media")
        return (
            "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend",
            {"userIds": [sender_staff_id]},
            robot_code,
        )

    async def _upload_dingtalk_media(self, media_path: str, media_type: str) -> str:
        token = await self._get_access_token()
        if not token:
            raise RuntimeError("Failed to obtain DingTalk access token")
        if not self._http_client:
            raise RuntimeError("HTTP client not initialized")

        path = self._dingtalk_media_path(media_path)
        if not path:
            raise RuntimeError("Invalid local media path")
        content_type = mimetypes.guess_type(path.name)[0] or "application/octet-stream"
        with path.open("rb") as media:
            response = await self._http_client.post(
                "https://oapi.dingtalk.com/media/upload",
                params={"access_token": token, "type": media_type},
                files={"media": (path.name, media, content_type)},
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

    def _dingtalk_media_path(self, media_path: str) -> Optional[Path]:
        safe_path = self.validate_media_delivery_path(media_path)
        if not safe_path:
            return None

        safe_root = os.environ.get("HERMES_WRITE_SAFE_ROOT", "").strip()
        if not safe_root:
            raise RuntimeError("HERMES_WRITE_SAFE_ROOT is required for DingTalk media")
        try:
            root = Path(safe_root).resolve(strict=True)
            path = Path(safe_path).resolve(strict=True)
            path.relative_to(root)
        except (OSError, ValueError):
            return None
        if not path.is_file():
            return None
        return path

    async def _send_dingtalk_media(
        self,
        destination: tuple[str, Dict[str, Any], str],
        msg_key: str,
        msg_param: Dict[str, Any],
        kind: str,
    ) -> SendResult:
        endpoint, target, robot_code = destination
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
                "msgKey": msg_key,
                "msgParam": json.dumps(msg_param),
                **target,
            },
            timeout=15.0,
        )
        if response.status_code >= 300:
            raise RuntimeError(
                f"DingTalk {kind} send returned HTTP {response.status_code}"
            )
        payload = response.json()
        if (
            payload.get("success") is False
            or payload.get("errcode") not in (None, 0, "0")
            or str(payload.get("code") or "").lower()
            not in ("", "0", "ok", "success")
        ):
            raise RuntimeError(f"DingTalk {kind} send rejected the request")
        message_id = str(
            payload.get("processQueryKey") or uuid.uuid4().hex[:12]
        )
        logger.info("[%s] %s sent to DingTalk", self.name, kind.title())
        return SendResult(success=True, message_id=message_id)

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
            path = self._dingtalk_media_path(image_path)
            if not path:
                return await self._dingtalk_media_failure(
                    chat_id, "Invalid local image path", metadata, "image"
                )

            if path.suffix.lower() not in {".jpg", ".jpeg", ".png", ".gif"}:
                return await self._dingtalk_media_failure(
                    chat_id, "Unsupported DingTalk image type", metadata, "image"
                )
            if path.stat().st_size > 20 * 1024 * 1024:
                return await self._dingtalk_media_failure(
                    chat_id, "DingTalk images must not exceed 20 MiB", metadata, "image"
                )

            destination = self._dingtalk_media_target(chat_id)
            media_id = await self._upload_dingtalk_media(str(path), "image")
            return await self._send_dingtalk_media(
                destination,
                "sampleImageMsg",
                {"photoURL": media_id},
                "image",
            )
        except RuntimeError as exc:
            error = str(exc)
            return await self._dingtalk_media_failure(
                chat_id, error, metadata, "image"
            )
        except Exception as exc:
            error = f"DingTalk image send failed ({type(exc).__name__})"
            return await self._dingtalk_media_failure(
                chat_id, error, metadata, "image"
            )

    async def _send_dingtalk_file(
        self,
        chat_id: str,
        file_path: str,
        file_name: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
        kind: str = "file",
    ) -> SendResult:
        try:
            path = self._dingtalk_media_path(file_path)
            if not path:
                return await self._dingtalk_media_failure(
                    chat_id, "Invalid local file path", metadata, kind
                )

            display_name = Path(file_name).name if file_name else path.name
            file_type = Path(display_name).suffix.lstrip(".").lower()
            if not file_type:
                return await self._dingtalk_media_failure(
                    chat_id, "DingTalk files must have an extension", metadata, kind
                )
            if path.stat().st_size > 20 * 1024 * 1024:
                return await self._dingtalk_media_failure(
                    chat_id, "DingTalk files must not exceed 20 MiB", metadata, kind
                )

            destination = self._dingtalk_media_target(chat_id)
            media_id = await self._upload_dingtalk_media(str(path), "file")
            return await self._send_dingtalk_media(
                destination,
                "sampleFile",
                {
                    "mediaId": media_id,
                    "fileName": display_name,
                    "fileType": file_type,
                },
                kind,
            )
        except RuntimeError as exc:
            error = str(exc)
            return await self._dingtalk_media_failure(
                chat_id, error, metadata, kind
            )
        except Exception as exc:
            error = f"DingTalk file send failed ({type(exc).__name__})"
            return await self._dingtalk_media_failure(
                chat_id, error, metadata, kind
            )

    async def send_document(
        self,
        chat_id: str,
        file_path: str,
        caption: Optional[str] = None,
        file_name: Optional[str] = None,
        reply_to: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
        **kwargs,
    ) -> SendResult:
        """Upload and send a local file through DingTalk Robot OpenAPI."""
        return await self._send_dingtalk_file(
            chat_id, file_path, file_name=file_name, metadata=metadata
        )

    async def send_voice(
        self,
        chat_id: str,
        audio_path: str,
        caption: Optional[str] = None,
        reply_to: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
        **kwargs,
    ) -> SendResult:
        """Send native DingTalk audio when metadata is available, else a file."""
        duration_ms = kwargs.get("duration_ms")
        try:
            path = self._dingtalk_media_path(audio_path)
            if duration_ms is None or not path or path.suffix.lower() not in {".ogg", ".amr"}:
                return await self._send_dingtalk_file(
                    chat_id, audio_path, metadata=metadata, kind="audio"
                )
            if path.stat().st_size > 20 * 1024 * 1024:
                return await self._dingtalk_media_failure(
                    chat_id, "DingTalk audio must not exceed 20 MiB", metadata, "audio"
                )
            destination = self._dingtalk_media_target(chat_id)
            media_id = await self._upload_dingtalk_media(str(path), "voice")
            return await self._send_dingtalk_media(
                destination,
                "sampleAudio",
                {"mediaId": media_id, "duration": str(int(duration_ms))},
                "audio",
            )
        except RuntimeError as exc:
            return await self._dingtalk_media_failure(
                chat_id, str(exc), metadata, "audio"
            )
        except Exception as exc:
            error = f"DingTalk audio send failed ({type(exc).__name__})"
            return await self._dingtalk_media_failure(
                chat_id, error, metadata, "audio"
            )

    async def send_video(
        self,
        chat_id: str,
        video_path: str,
        caption: Optional[str] = None,
        reply_to: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
        **kwargs,
    ) -> SendResult:
        """Send native DingTalk video when metadata is available, else a file."""
        duration_seconds = kwargs.get("duration_seconds")
        thumbnail_path = (
            kwargs.get("thumbnail_path")
            or kwargs.get("cover_path")
            or kwargs.get("cover_image_path")
        )
        try:
            path = self._dingtalk_media_path(video_path)
            if (
                duration_seconds is None
                or not thumbnail_path
                or not path
                or path.suffix.lower() != ".mp4"
            ):
                return await self._send_dingtalk_file(
                    chat_id, video_path, metadata=metadata, kind="video"
                )
            thumbnail = self._dingtalk_media_path(str(thumbnail_path))
            if not thumbnail or thumbnail.suffix.lower() not in {
                ".jpg", ".jpeg", ".png", ".gif",
            }:
                return await self._dingtalk_media_failure(
                    chat_id, "Invalid DingTalk video thumbnail", metadata, "video"
                )
            if (
                path.stat().st_size > 20 * 1024 * 1024
                or thumbnail.stat().st_size > 20 * 1024 * 1024
            ):
                return await self._dingtalk_media_failure(
                    chat_id, "DingTalk video media must not exceed 20 MiB", metadata, "video"
                )
            destination = self._dingtalk_media_target(chat_id)
            video_media_id = await self._upload_dingtalk_media(str(path), "video")
            cover_media_id = await self._upload_dingtalk_media(str(thumbnail), "image")
            return await self._send_dingtalk_media(
                destination,
                "sampleVideo",
                {
                    "duration": str(int(duration_seconds)),
                    "videoMediaId": video_media_id,
                    "videoType": "mp4",
                    "picMediaId": cover_media_id,
                },
                "video",
            )
        except RuntimeError as exc:
            return await self._dingtalk_media_failure(
                chat_id, str(exc), metadata, "video"
            )
        except Exception as exc:
            error = f"DingTalk video send failed ({type(exc).__name__})"
            return await self._dingtalk_media_failure(
                chat_id, error, metadata, "video"
            )
'''

v1_marker = "    # HERMES_DOCK_DINGTALK_IMAGE_PATCH_V1\n"
if v1_marker in text:
    start = text.index(v1_marker)
    end = text.index("    async def get_chat_info(", start)
    text = text[:start] + method_replacement + "\n" + text[end:]
else:
    if text.count(import_needle) != 1:
        raise RuntimeError("Hermes DingTalk import marker not found exactly once")
    if text.count(method_needle) != 1:
        raise RuntimeError("Hermes DingTalk media method marker not found exactly once")
    text = text.replace(import_needle, import_replacement, 1)
    text = text.replace(method_needle, method_replacement, 1)
target.write_text(text)
PY

"$PYTHON" -m py_compile "$TARGET"

actual_sha256=$(sha256sum "$TARGET" | awk '{print $1}')
if [ "$actual_sha256" != "$EXPECTED_V2_SHA256" ] || ! grep -q "$PATCH_MARKER" "$TARGET"; then
    echo "[dingtalk-media-patch] patched adapter fingerprint mismatch: $actual_sha256" >&2
    exit 1
fi

echo "[dingtalk-media-patch] applied"
