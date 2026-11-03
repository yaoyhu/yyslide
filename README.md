# yyslide

用 Markdown 写幻灯片，一条命令预览、生成 HTML 或导出 PDF。版式来自[蒋炎岩老师的课程幻灯片](https://jyywiki.cn/GSE/2026/slides1.html)，主题色为科大蓝。

单个可执行文件，模板、MathJax、highlight.js 和字体都已内置，无需 Python 或 Node。

## 安装

```bash
curl -fsSL https://raw.githubusercontent.com/yaoyhu/yyslide/main/install.sh | sh
```

或者 `go install github.com/yaoyhu/yyslide@latest`。Windows 从 [Releases](https://github.com/yaoyhu/yyslide/releases) 下载。

## 使用

```bash
yyslide new my-talk   # 新建工程，含一份演示全部写法的 slides.md
cd my-talk
yyslide preview       # 浏览器实时预览，保存即刷新
yyslide build         # 单文件 HTML，资源全部内联，可离线播放 → dist/my-talk.html
yyslide pdf           # 16:9 PDF → dist/my-talk.pdf
```

- 不指定文件时使用当前目录的 `slides.md`，也可以写 `yyslide pdf talk.md -o talk.pdf`。
- `yyslide <命令> -h` 查看选项，例如 `preview -host 0.0.0.0` 允许局域网访问。
- `yyslide pdf` 会在后台调用本机的 Chrome / Edge / Chromium 打印页面，以保证和播放效果一致。浏览器位置特殊时用 `YYSLIDE_CHROME=/path/to/chrome` 指定。

## 写法

```markdown
# 第一页的标题（首页自动使用封面版式）

---

# 每页开头的一级标题成为顶部标题栏

**粗体**显示为科大蓝，==标记==显示为红色，公式写 $E=mc^2$ 或 $$...$$，
美元符号写 \$。支持表格、任务列表、代码高亮、图片和内嵌 HTML。

---

<!-- cover -->
# 章节过渡页
```

单独一行 `---` 分页。图片路径相对于 Markdown 文件。

播放时用 `←` `→`、空格、`j` `k` 翻页，`f` 全屏。地址中的 `#/N` 对应第 N 页。

## 自定义样式

在工程目录运行 `yyslide eject` 得到 `template.html`，之后的命令都会使用它，直接修改其中的 CSS 即可（主色 `--blue: #004098`）。

## 开发

```bash
go test ./...
go build
```

推送 `v*` 标签后，GitHub Actions 会用 goreleaser 构建各平台二进制并发布到 Releases。

## 许可证

MIT。内置的 MathJax（Apache-2.0）、highlight.js（BSD-3-Clause）和 JetBrains Mono（OFL-1.1）遵循各自的许可证。
