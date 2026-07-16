# Word（python-docx）

依赖：`python-docx`，Python 中使用 `import docx`。先阅读 `../SKILL.md` 的容器路径、输出和安全规则。

## 目录

- [检查正文和表格](#检查正文和表格)
- [替换文本](#替换文本)
- [处理跨 run 文本](#处理跨-run-文本)
- [提取图片](#提取图片)
- [创建文档](#创建文档)
- [能力边界](#能力边界)

示例中的输入文件名只是占位符。用户指定了文件时，使用其在容器内的准确路径；只有批量任务才使用 `glob()`。

## 检查正文和表格

```python
# /// script
# requires-python = ">=3.10"
# dependencies = ["python-docx"]
# ///
from pathlib import Path
import docx

INPUT_PATH = Path("/opt/data/.dock/shared/input.docx")


def main():
    if not INPUT_PATH.is_file():
        raise FileNotFoundError(INPUT_PATH)

    document = docx.Document(str(INPUT_PATH))
    print(f"文件：{INPUT_PATH}")

    for index, paragraph in enumerate(document.paragraphs, start=1):
        text = paragraph.text.strip()
        if text:
            print(f"[P{index}] {text}")

    for table_index, table in enumerate(document.tables, start=1):
        print(f"\n=== 表格 {table_index}：{len(table.rows)} 行 x {len(table.columns)} 列 ===")
        for row_index, row in enumerate(table.rows, start=1):
            cells = [cell.text.replace("\n", " ").strip() for cell in row.cells]
            print(f"R{row_index}: " + " | ".join(cells))


if __name__ == "__main__":
    main()
```

读取标题结构时检查段落样式：

```python
for paragraph in document.paragraphs:
    style_name = paragraph.style.name or ""
    text = paragraph.text.strip()
    if text and (style_name.startswith("Heading") or style_name.startswith("标题")):
        print(f"[{style_name}] {text}")
```

## 替换文本

优先在单个 run 内替换，这能保留原 run 的字体、字号和强调格式。正文和表格必须分别遍历。

```python
# /// script
# requires-python = ">=3.10"
# dependencies = ["python-docx"]
# ///
from pathlib import Path
import docx

INPUT_PATH = Path("/opt/data/.dock/shared/input.docx")
OUTPUT_PATH = Path("/opt/data/.dock/shared/input_修改版.docx")
OLD_TEXT = "旧文本"
NEW_TEXT = "新文本"


def replace_in_paragraphs(paragraphs, old: str, new: str) -> int:
    count = 0
    for paragraph in paragraphs:
        for run in paragraph.runs:
            occurrences = run.text.count(old)
            if occurrences:
                run.text = run.text.replace(old, new)
                count += occurrences
    return count


def main():
    if not INPUT_PATH.is_file():
        raise FileNotFoundError(INPUT_PATH)
    if OUTPUT_PATH.exists():
        raise FileExistsError(f"输出文件已存在：{OUTPUT_PATH}")

    document = docx.Document(str(INPUT_PATH))
    count = replace_in_paragraphs(document.paragraphs, OLD_TEXT, NEW_TEXT)

    for table in document.tables:
        for row in table.rows:
            for cell in row.cells:
                count += replace_in_paragraphs(cell.paragraphs, OLD_TEXT, NEW_TEXT)

    if count == 0:
        raise ValueError(f"未找到要替换的文本：{OLD_TEXT}")

    document.save(str(OUTPUT_PATH))
    docx.Document(str(OUTPUT_PATH))
    print(f"替换 {count} 处，输出：{OUTPUT_PATH}")


if __name__ == "__main__":
    main()
```

## 处理跨 run 文本

Word 可能把一句话拆成多个 run。只有普通替换没有命中、且用户接受局部字符格式可能丢失时，才合并段落的 run：

```python
def replace_merged_runs(paragraph, old: str, new: str) -> int:
    if not paragraph.runs:
        return 0
    full_text = "".join(run.text for run in paragraph.runs)
    count = full_text.count(old)
    if count == 0:
        return 0
    paragraph.runs[0].text = full_text.replace(old, new)
    for run in paragraph.runs[1:]:
        run.text = ""
    return count
```

这个方法保留段落级对齐和缩进，但会把字符级格式集中到第一个 run。不要在复杂格式文档中默认使用。

## 提取图片

`.docx` 是 ZIP 包，原始图片位于 `word/media/`。提取文件不表示它们在正文中的出现顺序。

```python
# /// script
# requires-python = ">=3.10"
# dependencies = []
# ///
from pathlib import Path
import zipfile

INPUT_PATH = Path("/opt/data/.dock/shared/input.docx")
OUTPUT_DIR = Path("/opt/data/.dock/shared/input_图片")


def main():
    if not INPUT_PATH.is_file():
        raise FileNotFoundError(INPUT_PATH)
    if OUTPUT_DIR.exists():
        raise FileExistsError(f"输出目录已存在：{OUTPUT_DIR}")

    with zipfile.ZipFile(INPUT_PATH) as archive:
        media = sorted(name for name in archive.namelist() if name.startswith("word/media/"))
        if not media:
            raise ValueError("文档中没有可提取的图片")
        OUTPUT_DIR.mkdir(parents=True)
        for name in media:
            target = OUTPUT_DIR / Path(name).name
            target.write_bytes(archive.read(name))

    extracted = list(OUTPUT_DIR.iterdir())
    if len(extracted) != len(media):
        raise RuntimeError("图片提取数量校验失败")
    print(f"已提取 {len(extracted)} 张图片到：{OUTPUT_DIR}")


if __name__ == "__main__":
    main()
```

## 创建文档

```python
# /// script
# requires-python = ">=3.10"
# dependencies = ["python-docx"]
# ///
from pathlib import Path
import docx

OUTPUT_PATH = Path("/opt/data/.dock/shared/项目汇报.docx")


def main():
    if OUTPUT_PATH.exists():
        raise FileExistsError(f"输出文件已存在：{OUTPUT_PATH}")

    document = docx.Document()
    document.add_heading("项目汇报", level=1)
    document.add_paragraph("这里是正文第一段。")
    document.add_heading("一、项目概况", level=2)

    table = document.add_table(rows=3, cols=2)
    table.style = "Table Grid"
    rows = [["名称", "数值"], ["事项 A", "100"], ["事项 B", "200"]]
    for row_index, row_data in enumerate(rows):
        for column_index, value in enumerate(row_data):
            table.rows[row_index].cells[column_index].text = value

    document.save(str(OUTPUT_PATH))
    validated = docx.Document(str(OUTPUT_PATH))
    if not validated.paragraphs or len(validated.tables) != 1:
        raise RuntimeError("输出文档结构校验失败")
    print(f"已创建：{OUTPUT_PATH}")


if __name__ == "__main__":
    main()
```

## 能力边界

- `document.paragraphs` 不包含页眉、页脚、文本框和部分嵌套对象；需要这些内容时单独遍历对应结构。
- 跨 run 替换可能损失字符级格式，先使用普通替换。
- 复杂域、目录和某些链接写入后可能需要在 Word 中刷新。
- 不修改含宏、密码保护或依赖复杂嵌入对象的文档。
