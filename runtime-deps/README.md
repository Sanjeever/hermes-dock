# Runtime dependency wheelhouses

Hermes Dock embeds one CPython 3.13 Linux wheelhouse in each desktop build:

- Windows amd64 and Linux amd64 embed `linux-amd64`.
- macOS arm64 embeds `linux-arm64`.

The wheels are tracked with Git LFS. Runtime installation is strictly offline;
the container init helpers must not fall back to a package index or direct URL.
`SHA256SUMS` is generated from the exact files in each architecture directory
and is verified before the helpers install anything.

Top-level dependency inputs live in `requirements/`. Architecture-specific
lock files and their complete transitive wheel sets live under the corresponding
platform directory. The Feishu and DingTalk helpers intentionally resolve their
pinned top-level packages against the local wheel set so compatible packages
already provided by the shared Hermes environment are retained. PaddleOCR and
ddddocr are installed on demand by the bundled `image-text-ocr` and
`captcha-ocr` skills into separate `data/.dock` environments and are not part
of these wheelhouses. Interrupted extraction directories are removed before
the next extraction. An obsolete bundle is removed only after Hermes has
successfully started with the current version.
