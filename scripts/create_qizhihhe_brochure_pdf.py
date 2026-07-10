# /// script
# requires-python = ">=3.10"
# dependencies = ["reportlab", "pypdf"]
# ///
from pathlib import Path

from pypdf import PdfReader
from reportlab.lib.colors import Color, HexColor
from reportlab.lib.pagesizes import A4
from reportlab.lib.utils import ImageReader
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.cidfonts import UnicodeCIDFont
from reportlab.pdfgen import canvas


ROOT = Path(__file__).resolve().parents[1]
OUTPUT_PATH = ROOT / "企智盒产品宣传手册.pdf"
LOGO_PATH = ROOT / "build" / "appicon.png"

PAGE_WIDTH, PAGE_HEIGHT = A4
MARGIN = 52

INK = HexColor("#20201E")
MUTED = HexColor("#6F6A64")
PAPER = HexColor("#FFFDF9")
WARM = HexColor("#FFF1E5")
PALE = HexColor("#FFE1CC")
ORANGE = HexColor("#FF7043")
RED = HexColor("#EF4436")
DARK_RED = HexColor("#A72E2A")
WHITE = HexColor("#FFFFFF")


def register_fonts() -> None:
    pdfmetrics.registerFont(UnicodeCIDFont("STSong-Light"))


def font_name() -> str:
    return "STSong-Light"


def text_width(text: str, size: float) -> float:
    return pdfmetrics.stringWidth(text, font_name(), size)


def wrap(text: str, size: float, width: float) -> list[str]:
    lines: list[str] = []
    line = ""
    for char in text:
        if char == "\n":
            if line:
                lines.append(line)
            line = ""
            continue
        if line and text_width(line + char, size) > width:
            lines.append(line)
            line = char
        else:
            line += char
    if line:
        lines.append(line)
    return lines


def draw_text(
    c: canvas.Canvas,
    text: str,
    x: float,
    y: float,
    size: float,
    color: Color = INK,
    max_width: float | None = None,
    leading: float | None = None,
) -> float:
    leading = leading or size * 1.55
    lines = wrap(text, size, max_width) if max_width else text.split("\n")
    c.setFont(font_name(), size)
    c.setFillColor(color)
    for line in lines:
        c.drawString(x, y, line)
        y -= leading
    return y


def rounded_rect(c: canvas.Canvas, x: float, y: float, w: float, h: float, fill: Color, radius: float = 18) -> None:
    c.setFillColor(fill)
    c.roundRect(x, y, w, h, radius, fill=1, stroke=0)


def page_base(c: canvas.Canvas, number: str, section: str) -> None:
    c.setFillColor(PAPER)
    c.rect(0, 0, PAGE_WIDTH, PAGE_HEIGHT, fill=1, stroke=0)
    c.setFillColor(WARM)
    c.circle(PAGE_WIDTH + 35, PAGE_HEIGHT - 25, 116, fill=1, stroke=0)
    c.setFillColor(PALE)
    c.circle(-36, -32, 98, fill=1, stroke=0)
    c.setFillColor(ORANGE)
    c.roundRect(MARGIN, PAGE_HEIGHT - 84, 35, 7, 3.5, fill=1, stroke=0)
    draw_text(c, number, MARGIN, PAGE_HEIGHT - 115, 11, RED)
    draw_text(c, section, MARGIN, PAGE_HEIGHT - 145, 26, INK)
    draw_text(c, "企智盒", PAGE_WIDTH - MARGIN - 48, 36, 10, MUTED)


def bullet(c: canvas.Canvas, text: str, x: float, y: float, width: float, size: float = 14) -> float:
    c.setFillColor(RED)
    c.circle(x + 4, y + 5, 4, fill=1, stroke=0)
    return draw_text(c, text, x + 20, y, size, INK, width - 20, size * 1.65)


def save_page(c: canvas.Canvas) -> None:
    c.showPage()


def cover(c: canvas.Canvas, logo: ImageReader) -> None:
    c.setFillColor(PAPER)
    c.rect(0, 0, PAGE_WIDTH, PAGE_HEIGHT, fill=1, stroke=0)
    c.setFillColor(WARM)
    c.circle(PAGE_WIDTH - 12, PAGE_HEIGHT - 28, 185, fill=1, stroke=0)
    c.setFillColor(PALE)
    c.circle(48, 74, 146, fill=1, stroke=0)
    c.setFillColor(ORANGE)
    c.roundRect(MARGIN, PAGE_HEIGHT - 122, 42, 8, 4, fill=1, stroke=0)
    c.drawImage(logo, PAGE_WIDTH - 255, PAGE_HEIGHT - 330, 214, 214, mask="auto", preserveAspectRatio=True)
    draw_text(c, "企智盒：把 AI", MARGIN, 300, 38, INK)
    draw_text(c, "放进企业日常工作", MARGIN, 238, 38, INK)
    c.setFillColor(RED)
    c.roundRect(MARGIN, 174, 88, 6, 3, fill=1, stroke=0)
    draw_text(c, "企智盒", MARGIN, 62, 12, MUTED)
    save_page(c)


def what_it_is(c: canvas.Canvas) -> None:
    page_base(c, "01", "企智盒是什么")
    draw_text(c, "很多企业已经用过豆包。", MARGIN, 590, 23, INK)
    draw_text(c, "但 AI 往往只停在“问一问”。", MARGIN, 545, 16, MUTED)
    draw_text(c, "企智盒帮助企业把 AI 用进每天的沟通和工作。", MARGIN, 462, 25, INK, 470, 38)
    cards = [
        ("常用工具", "接入微信、企业微信和飞书。"),
        ("明确岗位", "按岗位设置资料和做事规则。"),
        ("本地使用", "部署在企业指定的本地设备上。"),
    ]
    y = 347
    for index, (heading, body) in enumerate(cards):
        rounded_rect(c, MARGIN, y - index * 94, 491, 72, WHITE, 16)
        c.setFillColor(ORANGE)
        c.circle(MARGIN + 26, y + 35 - index * 94, 12, fill=1, stroke=0)
        draw_text(c, heading, MARGIN + 55, y + 39 - index * 94, 15, INK)
        draw_text(c, body, MARGIN + 55, y + 14 - index * 94, 12, MUTED)
    draw_text(c, "模型服务由企业选择。", MARGIN, 77, 13, MUTED)
    save_page(c)


def comparison(c: canvas.Canvas) -> None:
    page_base(c, "02", "不只是聊天，更是日常协助")
    col_gap = 18
    col_w = (PAGE_WIDTH - 2 * MARGIN - col_gap) / 2
    left_x = MARGIN
    right_x = MARGIN + col_w + col_gap
    rounded_rect(c, left_x, 236, col_w, 360, HexColor("#F3F1EE"), 22)
    rounded_rect(c, right_x, 236, col_w, 360, HexColor("#FFF0E6"), 22)
    draw_text(c, "用豆包", left_x + 27, 548, 22, INK)
    draw_text(c, "有问题时，", left_x + 27, 476, 17, INK)
    draw_text(c, "打开应用，再提问。", left_x + 27, 442, 17, INK)
    draw_text(c, "用企智盒", right_x + 27, 548, 22, RED)
    draw_text(c, "把 AI 放进企业", right_x + 27, 476, 17, INK)
    draw_text(c, "常用的沟通工具里。", right_x + 27, 442, 17, INK)
    draw_text(c, "设置清楚的岗位、资料和规则。", right_x + 27, 373, 13, MUTED, col_w - 54, 23)
    draw_text(c, "长期协助一类具体工作。", right_x + 27, 306, 13, MUTED)
    rounded_rect(c, MARGIN, 115, PAGE_WIDTH - 2 * MARGIN, 76, INK, 18)
    draw_text(c, "豆包适合随时问一问。", MARGIN + 24, 157, 16, WHITE)
    draw_text(c, "企智盒适合长期协助一个明确岗位。", MARGIN + 24, 130, 16, WHITE)
    save_page(c)


def assistants(c: canvas.Canvas) -> None:
    page_base(c, "03", "三个常见助手")
    cards = [
        ("销售跟进助手", ["整理客户需求。", "起草跟进信息。", "快速准备产品介绍和回复。"]),
        ("客户咨询助手", ["根据已确认的资料回复。", "帮助客服更快找到信息。"]),
        ("内部资料助手", ["查询制度、产品资料和培训内容。", "少问人。少翻文件。"]),
    ]
    y_positions = [520, 346, 172]
    for index, ((title, lines), y) in enumerate(zip(cards, y_positions)):
        fill = WHITE if index != 1 else HexColor("#FFF0E6")
        rounded_rect(c, MARGIN, y, PAGE_WIDTH - 2 * MARGIN, 138, fill, 20)
        c.setFillColor(ORANGE if index != 1 else RED)
        c.circle(MARGIN + 31, y + 100, 14, fill=1, stroke=0)
        draw_text(c, title, MARGIN + 62, y + 98, 20, INK)
        text_y = y + 62
        for line in lines:
            text_y = bullet(c, line, MARGIN + 62, text_y, PAGE_WIDTH - 2 * MARGIN - 90, 13) - 3
    draw_text(c, "重要回复、报价、承诺和投诉，建议由员工确认后再发出。", MARGIN, 96, 11, MUTED, PAGE_WIDTH - 2 * MARGIN, 17)
    save_page(c)


def start(c: canvas.Canvas) -> None:
    page_base(c, "04", "开始使用，不需要先懂技术")
    steps = [
        ("1", "先聊清楚要解决什么", "预约演示或上门评估。一起确定先做哪个岗位。"),
        ("2", "准备首批资料", "提供产品资料、常见问答、制度、话术或案例。"),
        ("3", "完成部署和配置", "完成基础部署、平台接入和配置。"),
        ("4", "开始使用", "完成管理员培训。再逐步补充资料和规则。"),
    ]
    y = 556
    for number, title, body in steps:
        c.setFillColor(RED)
        c.circle(MARGIN + 20, y + 9, 20, fill=1, stroke=0)
        c.setFillColor(WHITE)
        c.setFont(font_name(), 16)
        c.drawCentredString(MARGIN + 20, y + 3, number)
        draw_text(c, title, MARGIN + 58, y + 12, 18, INK)
        draw_text(c, body, MARGIN + 58, y - 19, 12, MUTED, 410, 19)
        if number != "4":
            c.setStrokeColor(PALE)
            c.setLineWidth(2)
            c.line(MARGIN + 20, y - 50, MARGIN + 20, y - 93)
        y -= 142
    save_page(c)


def options(c: canvas.Canvas) -> None:
    page_base(c, "05", "两种方式使用企智盒")
    col_gap = 18
    col_w = (PAGE_WIDTH - 2 * MARGIN - col_gap) / 2
    panels = [
        (MARGIN, HexColor("#FFF0E6"), "企智盒一体机", "提供预装环境的专用迷你主机。\n\n适合希望少折腾、\n尽快开始使用的企业。"),
        (MARGIN + col_w + col_gap, HexColor("#F4F1EE"), "企智盒本地部署服务", "部署在企业现有的合适设备上。\n\n适合已有设备、\n不需要新硬件的企业。"),
    ]
    for x, fill, title, body in panels:
        rounded_rect(c, x, 245, col_w, 344, fill, 24)
        c.setFillColor(ORANGE if x == MARGIN else RED)
        c.circle(x + 39, 537, 17, fill=1, stroke=0)
        draw_text(c, title, x + 28, 483, 21, INK, col_w - 56, 30)
        draw_text(c, body, x + 28, 397, 14, MUTED, col_w - 56, 25)
    rounded_rect(c, MARGIN, 127, PAGE_WIDTH - 2 * MARGIN, 67, INK, 17)
    draw_text(c, "两种方式都包含基础部署、配置和培训。", MARGIN + 24, 153, 15, WHITE)
    save_page(c)


def suitable(c: canvas.Canvas) -> None:
    page_base(c, "06", "适合这样的企业")
    draw_text(c, "企智盒适合先从一个岗位开始。", MARGIN, 590, 23, INK)
    items = [
        "企业有稳定业务。",
        "员工大约 10 到 100 人。",
        "希望 AI 帮助销售、客服、运营或内部协作。",
        "没有专职的 AI 或 IT 团队。",
        "愿意提供首批资料，并安排一位业务对接人。",
    ]
    y = 518
    for item in items:
        y = bullet(c, item, MARGIN, y, PAGE_WIDTH - 2 * MARGIN, 15) - 17
    rounded_rect(c, MARGIN, 128, PAGE_WIDTH - 2 * MARGIN, 83, HexColor("#FFF0E6"), 20)
    draw_text(c, "先用出效果。", MARGIN + 24, 173, 18, RED)
    draw_text(c, "再增加更多助手。", MARGIN + 24, 144, 18, INK)
    save_page(c)


def contact(c: canvas.Canvas, logo: ImageReader) -> None:
    c.setFillColor(PAPER)
    c.rect(0, 0, PAGE_WIDTH, PAGE_HEIGHT, fill=1, stroke=0)
    c.setFillColor(WARM)
    c.circle(PAGE_WIDTH - 20, PAGE_HEIGHT - 25, 160, fill=1, stroke=0)
    c.setFillColor(PALE)
    c.circle(12, 34, 120, fill=1, stroke=0)
    c.drawImage(logo, PAGE_WIDTH - 255, PAGE_HEIGHT - 315, 190, 190, mask="auto", preserveAspectRatio=True)
    draw_text(c, "想了解企智盒", MARGIN, 352, 34, INK)
    draw_text(c, "是否适合您的企业？", MARGIN, 296, 34, INK)
    c.setFillColor(RED)
    c.roundRect(MARGIN, 234, 92, 6, 3, fill=1, stroke=0)
    draw_text(c, "请联系为您提供手册的顾问，", MARGIN, 175, 17, MUTED)
    draw_text(c, "预约演示或上门评估。", MARGIN, 141, 17, MUTED)
    draw_text(c, "企智盒", MARGIN, 62, 12, MUTED)
    save_page(c)


def main() -> None:
    if not LOGO_PATH.exists():
        raise FileNotFoundError(LOGO_PATH)
    register_fonts()
    c = canvas.Canvas(str(OUTPUT_PATH), pagesize=A4, pageCompression=1)
    c.setTitle("企智盒产品宣传手册")
    c.setAuthor("企智盒")
    logo = ImageReader(str(LOGO_PATH))
    cover(c, logo)
    what_it_is(c)
    comparison(c)
    assistants(c)
    start(c)
    options(c)
    suitable(c)
    contact(c, logo)
    c.save()

    reader = PdfReader(str(OUTPUT_PATH))
    if len(reader.pages) != 8:
        raise RuntimeError(f"页数异常：{len(reader.pages)}")
    for page in reader.pages:
        width = float(page.mediabox.width)
        height = float(page.mediabox.height)
        if round(width, 1) != round(PAGE_WIDTH, 1) or round(height, 1) != round(PAGE_HEIGHT, 1):
            raise RuntimeError(f"纸张尺寸异常：{width} × {height}")
    print(f"已生成：{OUTPUT_PATH}")
    print(f"页数：{len(reader.pages)}；纸张：A4 竖版")


if __name__ == "__main__":
    main()
