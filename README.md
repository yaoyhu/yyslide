# JYY Slides Template（科大蓝版）

基于[蒋炎岩老师课程幻灯片页面](https://jyywiki.cn/GSE/2026/slides1.html)提取的可复用模板。保留标题栏、翻页动画、自动缩放、代码高亮和数学公式效果，将主题色改为科大蓝，并增加 Markdown 构建流程。

日常使用时只需维护 Markdown，不需要逐页手写 HTML，也不需要启动 Web 服务。完整的编写示例请直接查看 [`slides-template.md`](slides-template.md)，浏览器效果见 [`slides-template.html`](slides-template.html)。

## 项目结构

```text
jyy-slides-template/
├── slides-template.md        # 详细 Markdown 编写示例
├── slides-template.html      # 对应的可直接播放示例
├── build.py                  # Markdown → HTML 构建脚本
├── templates/
│   └── template.html         # 页面结构、主题样式和浏览器端交互
├── images/
│   └── example.svg           # 示例图片资源
├── vendor/
│   ├── mathjax-tex-svg.js    # 本地 MathJax，用于公式渲染
│   ├── highlight.min.js      # 本地 highlight.js，用于代码高亮
│   └── fonts/                # JetBrains Mono 等宽字体（OFL 授权）
├── requirements.txt          # Python 构建依赖
├── .gitignore                # 排除本地环境和系统文件
└── README.md                 # 架构、配置、构建与发布说明
```

`.venv/` 是本地构建环境，已被 Git 忽略，不需要提交或分发。

## 架构与工作流程

### 1. 构建期：Markdown 生成完整页面

```text
slides-template.md
        │
        │  build.py + Mistune
        │  · 按独立一行 --- 分页
        │  · Markdown 转 HTML
        │  · 保护并恢复 TeX 公式
        │  · 将各页内容编码为 JavaScript 数据
        ▼
templates/template.html ─────▶ slides-template.html
```

`build.py` 读取 `templates/template.html`，替换其中 `__SLIDES_DATA_START__` 和 `__SLIDES_DATA_END__` 标记之间的示例数据，同时设置 HTML 标题。模板本身不被修改，因此可以反复构建不同内容。

Python 和 Mistune 只在构建时需要；打开生成后的 HTML 不需要安装 Python。

### 2. 运行期：浏览器负责展示

生成的 HTML 在浏览器中完成以下工作：

- 将每页开头的 `<h1>` 提取为独立顶部标题栏，其余内容放入正文区域。
- 根据窗口、标题和正文尺寸自动缩放内容，并处理图片与公式加载后的重新排版。
- 提供前后翻页、首尾跳转、入场/离场动画和全屏播放。
- 使用本地 highlight.js 高亮代码，使用本地 MathJax 渲染 TeX 公式。

- 第一页自动使用封面版式（整页科大蓝、标题居中）；其他页在开头写一行 `<!-- cover -->` 即可作为章节过渡页。
- 底部进度条和右下角页码显示播放进度。

第三方运行库和代码字体均已下载到 `vendor/`，正常文字、代码和公式均可离线显示，不依赖 CDN。

强调写法约定：`**粗体**` 显示为科大蓝，`==标记==` 显示为红色醒目标记，用于少量需要警示的内容。

## 构建环境

### 方式一：交给 AI Agent 一键配置（推荐）

将下面整段提示词发给你使用的 AI Agent：

```text
请为当前项目配置并验证 Markdown 幻灯片构建环境：
1. 优先使用 uv；如果当前系统没有 uv，请先安装并确认 uv 可用。
2. 在项目根目录创建 .venv，根据 requirements.txt 安装依赖，不要修改系统 Python 环境。
3. 执行 uv run python build.py slides-template.md，确认能生成 slides-template.html。
4. 告诉我如何打开生成的 HTML。
除安装 uv、创建本地虚拟环境和生成示例 HTML 外，不要修改项目源文件。
```

Agent 完成配置后，以后修改 Markdown 并重新构建即可，无需重复安装环境。

### 方式二：手动配置

安装 [uv](https://docs.astral.sh/uv/) 后，在项目目录执行：

```bash
uv venv
uv pip install -r requirements.txt
```

uv 命令会自动使用项目中的 `.venv/`，不需要手动激活虚拟环境。

## 构建与播放

在项目目录执行：

```bash
uv run python build.py slides-template.md
open slides-template.html
```

默认输出到与输入文件同目录、同名的 `.html` 文件；再次构建会覆盖该输出文件。需要长期修改的内容应保存在 Markdown 中，而不是生成后的 HTML 中。

也可以指定输入文件、输出文件和浏览器标签页标题：

```bash
uv run python build.py my-talk.md -o my-talk.html --title "课程报告"
open my-talk.html
```

Linux 或 Windows 可直接用系统浏览器打开生成的 HTML，或在文件管理器中双击。

## 播放操作

| 操作 | 按键 |
| --- | --- |
| 下一页 | `Space`、`→`、`↓`、`Enter`、`PageDown`、`j`、`l` |
| 上一页 | `←`、`↑`、`Backspace`、`PageUp`、`h`、`k` |
| 第一页 | `Home` |
| 最后一页 | `End` |
| 进入/退出全屏 | `f` |

页面右下角显示当前页码和总页数。浏览器通常要求先点击页面或进行一次键盘交互，才允许进入全屏。

## 修改模板

- **修改主题和排版**：编辑 `templates/template.html` 中的 CSS。科大蓝主色定义为 `--blue: #004098`，标题栏等位置也有对应颜色设置。
- **修改翻页或缩放行为**：编辑 `templates/template.html` 中的浏览器端 JavaScript。
- **修改分页、Markdown 或公式处理**：编辑 `build.py`。
- **学习内容写法**：编辑或复制 `slides-template.md`，其中包含详细示例。

修改模板后，需要重新运行 `build.py`，已有输出文件才会更新。

## 分享与迁移

如果只需要播放成品，复制以下内容，并保持相对目录不变：

```text
my-talk.html
vendor/
images/       # 使用了其中的本地图片时需要
```

如果还需要继续编辑和重新构建，建议克隆完整仓库，再按“构建环境”一节安装依赖。

由于资源使用相对路径，若将输出文件生成到其他目录，也要确保该目录旁存在对应的 `vendor/` 和图片资源，或改用合适的资源路径。
