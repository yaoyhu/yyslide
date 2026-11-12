# yyslide

用 Markdown 写幻灯片，一条命令预览、生成 HTML 或导出 PDF。版式来自[蒋炎岩老师的课程幻灯片](https://jyywiki.cn/GSE/2026/slides1.html)，主题色为科大蓝。

单个可执行文件，模板、MathJax、highlight.js、Pele 和字体都已内置，无需 Python 或 Node。

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

````markdown
# 第一页的标题（首页自动使用封面版式）

---

# 每页开头的一级标题成为顶部标题栏

**粗体**显示为科大蓝，==标记==显示为红色，公式写 $E=mc^2$ 或 $$...$$，
美元符号写 \$。支持表格、任务列表、代码高亮、图片和内嵌 HTML。

---

# 脚注

正文中标注引用[^1]，脚注自动显示在本页底部。

[^1]: 定义要和引用写在同一页，序号每页从 1 开始。

---

# Mermaid 图表

```mermaid
flowchart LR
  写作 --> 预览 --> 导出
```

---

<!-- section -->
# 章节过渡页

---

<!-- image-right -->
# 图片与标题

文字留在左栏，图片自动移到右侧。

![](images/figure.png)
````

单独一行 `---` 分页。图片路径相对于 Markdown 文件。

在一页最开头写标记可以选择版式：

| 标记 | 版式 |
| --- | --- |
| `<!-- cover -->` | 标题幻灯片：标题居中，第一页默认使用 |
| `<!-- section -->` | 节标题：标题靠左，用于章节过渡 |
| `<!-- agenda -->` | 目录：自动列出所有 `section` 页并高亮当前章节 |
| `<!-- two-cols -->` | 两栏：单独一行 `::col::` 分隔各栏 |
| `<!-- three-cols -->` | 三栏：同上 |
| `<!-- compare -->` | 比较：两栏，栏首小标题成为表头 |
| `<!-- image-right -->` `<!-- image-left -->` | 图片与标题：正文里的图片移到一侧，完整显示 |
| `<!-- full -->` | 全屏图：图片铺满整页，标题和文字浮在左下角 |
| `<!-- figure -->` | 结论 + 大图：标题写结论，图片占满其余高度，alt 文字作图注 |
| `<!-- results -->` | 结果表 + 结论：首列加粗的行高亮，引用块作为结论条 |
| `<!-- statement -->` | 一句话观点：一级标题写观点 |
| `<!-- fact -->` | 大数字：一级标题写数字 |
| `<!-- quote -->` | 引用：引用块放大，后面的段落作为出处 |
| `<!-- end -->` | 总结 / 结束：最后一段弱化，用于致谢 |

`image-right`、`image-left` 和 `full` 可以加 `crop`（如 `<!-- image-right crop -->`），让图片铺满并居中裁掉超出部分，适合白边很大的图。

在页首单独写一行 `<!-- hide -->` 可以隐藏这一页，方便在一份文件里保留旧页面、按场合挑选。它可以和版式标记各占一行、顺序不限。隐藏页不会出现在预览、HTML 和 PDF 中，页码和目录按剩下的页面计算；`build` 和 `pdf` 会提示隐藏了几页。

Mermaid 图由 [Pele](https://github.com/obsidianmd/pele) 渲染，语法兼容 Mermaid，布局和样式是它自己的；配色跟随主题，`%%{init: ...}%%` 中的主题设置不生效。语法错误会以红字显示在图的位置，`yyslide pdf` 同时在终端给出警告。

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

MIT。内置的 MathJax（Apache-2.0）、highlight.js（BSD-3-Clause）、Pele（MIT）和 JetBrains Mono（OFL-1.1）遵循各自的许可证。
