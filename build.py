#!/usr/bin/env python3
import argparse
import html
import json
import re
from pathlib import Path

try:
    import mistune
except ImportError:
    raise SystemExit("缺少 Mistune，请先运行：python3 -m pip install -r requirements.txt")


def split_slides(source):
    pages = []
    current = []
    fence = None
    for line in source.splitlines():
        marker = re.match(r"^[ \t]*(`{3,}|~{3,})", line)
        if marker:
            token = marker.group(1)
            if fence is None:
                fence = token[0], len(token)
            elif token[0] == fence[0] and len(token) >= fence[1]:
                fence = None
            current.append(line)
        elif fence is None and re.fullmatch(r"[ \t]*---[ \t]*", line):
            page = "\n".join(current).strip()
            if page:
                pages.append(page)
            current = []
        else:
            current.append(line)
    page = "\n".join(current).strip()
    if page:
        pages.append(page)
    return pages


def render_slide(markdown, source):
    formulas = []

    def stash_formula(match):
        formulas.append(match.group(0))
        return f"@@SLIDES_MATH_{len(formulas) - 1}@@"

    protected = re.sub(r"\$\$[\s\S]+?\$\$|\$(?:\\.|[^$\n])+\$", stash_formula, source)
    rendered = markdown(protected).strip()
    for index, formula in enumerate(formulas):
        rendered = rendered.replace(f"@@SLIDES_MATH_{index}@@", html.escape(formula, quote=False))
    return rendered


def main():
    parser = argparse.ArgumentParser(description="将 Markdown 构建为离线 HTML 幻灯片")
    parser.add_argument("input", type=Path, help="Markdown 文件，使用单独一行 --- 分页")
    parser.add_argument("-o", "--output", type=Path, help="输出文件，默认与输入同名")
    parser.add_argument("--title", help="HTML 标题，默认使用输入文件名")
    args = parser.parse_args()

    source_path = args.input.resolve()
    output_path = args.output.resolve() if args.output else source_path.with_suffix(".html")
    template_path = Path(__file__).parent / "templates" / "template.html"
    markdown = mistune.create_markdown(
        escape=False,
        plugins=["strikethrough", "mark", "table", "task_lists"],
    )
    slides = [render_slide(markdown, page) for page in split_slides(source_path.read_text(encoding="utf-8"))]
    if not slides:
        parser.error("Markdown 中没有幻灯片内容")

    payload = json.dumps(slides, ensure_ascii=False)
    payload = payload.replace("&", "\\u0026").replace("<", "\\u003c").replace(">", "\\u003e")
    payload = payload.replace("\u2028", "\\u2028").replace("\u2029", "\\u2029")

    template = template_path.read_text(encoding="utf-8")
    pattern = r"(?<=// __SLIDES_DATA_START__\n)[\s\S]*?(?=// __SLIDES_DATA_END__)"
    replacement = f"const MY_SLIDES = {payload};\n"
    result, count = re.subn(pattern, lambda _: replacement, template, count=1)
    if count != 1:
        raise SystemExit("templates/template.html 中缺少幻灯片数据标记")

    title = html.escape(args.title or source_path.stem)
    result = re.sub(r"<title>[\s\S]*?</title>", f"<title>{title}</title>", result, count=1)
    output_path.write_text(result, encoding="utf-8")
    print(f"已生成 {len(slides)} 页：{output_path}")


if __name__ == "__main__":
    main()
