---
name: captcha-ocr
description: Recognize simple static character or arithmetic image CAPTCHAs from local element screenshots with a pinned ddddocr beta model and a hash-locked on-demand runtime. Use only for ordinary login CAPTCHA images made of a short sequence of letters, digits, Chinese characters, or basic arithmetic symbols. Do not use for sliders, click-in-order challenges, rotated puzzles, reCAPTCHA, hCaptcha, Cloudflare challenges, device verification, or other anti-automation mechanisms.
---

# CAPTCHA OCR

Use the local CAPTCHA-specific model on one element screenshot:

```bash
/opt/hermes/.venv/bin/python \
  skills/productivity/captcha-ocr/scripts/run_ocr.py \
  /opt/data/path/to/captcha.png
```

Always use `run_ocr.py`; do not invoke `captcha_ocr.py` directly, import `ddddocr` ad hoc, or install packages manually. On first use, the wrapper creates `/opt/data/.dock/captcha-ocr-venv` and downloads hash-locked binary wheels. The model is contained in the pinned `ddddocr` wheel; recognition does not download a model or send the image to an external service. Later runs reuse the local environment.

The script prints one JSON object:

- `success`: whether recognition completed.
- `textFound`: whether the model returned non-empty text.
- `text`: the recognized characters or expression.
- `model`: the model name, version, and variant.

Treat `success: true` with `textFound: false` as an unrecognized image. Treat `success: false` as an execution failure and preserve its `error`; do not invent missing characters.

Only accept output that matches the CAPTCHA's visible position, expected length, and stated character type. Remove surrounding whitespace only. Do not guess, autocorrect, or add characters. For a basic arithmetic CAPTCHA, validate the complete expression and calculate it according to the calling login workflow before filling the answer.

Accept one local raster image under the current `HERMES_DOCK_PROFILE_HOME`, at most 8 MiB and 4096 pixels on either side. Reject the request if the current profile home is unavailable. Use a fresh CAPTCHA element screenshot after every refresh. Do not pass a full-page screenshot, reuse an old result after the CAPTCHA changes, or extract image bytes from page source, canvas, blob, base64, or network requests.

Even though the upstream library includes other CAPTCHA functions, never use this skill for sliders, click targets, rotation or puzzle challenges, reCAPTCHA, hCaptcha, Cloudflare challenges, behavior checks, or any other anti-automation mechanism. Stop and require the user to complete those challenges.
