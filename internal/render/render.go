// Package render 把 Markdown 幻灯片渲染成可播放的 HTML 页面。
package render

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	ghtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/yaoyhu/yyslide/assets"
)

// TemplateFile 是 deck 目录中可选的自定义模板（由 `yyslide eject` 生成），存在时优先于内置模板。
const TemplateFile = "template.html"

// Deck 是一份已渲染的幻灯片。
type Deck struct {
	Source string   // Markdown 文件绝对路径
	Dir    string   // Markdown 所在目录，图片等相对路径以此为基准
	Title  string   // 页面标题
	Slides []string // 每页渲染后的 HTML 片段，不含隐藏页
	Hidden int      // 用 <!-- hide --> 隐藏的页数
}

// Load 读取并渲染 Markdown 文件。title 为空时使用首页一级标题，再退回文件名。
func Load(path, title string) (*Deck, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	src, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	slides, hidden := renderSource(string(src))
	if len(slides) == 0 && hidden > 0 {
		return nil, fmt.Errorf("%s 中的 %d 页都已用 <!-- hide --> 隐藏", path, hidden)
	}
	if len(slides) == 0 {
		return nil, fmt.Errorf("%s 中没有幻灯片内容", path)
	}
	if title == "" {
		title = firstHeading(slides[0])
	}
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
	}
	return &Deck{Source: abs, Dir: filepath.Dir(abs), Title: title, Slides: slides, Hidden: hidden}, nil
}

// Slides 按独立一行 --- 分页（代码块内除外），去掉隐藏页后逐页转换为 HTML。
func Slides(source string) []string {
	slides, _ := renderSource(source)
	return slides
}

func renderSource(source string) ([]string, int) {
	pages, hidden := visible(split(source))
	md := newMarkdown()
	out := make([]string, len(pages))
	for i, page := range pages {
		out[i] = renderPage(md, page)
	}
	return out, hidden
}

// visible 去掉页首带 <!-- hide --> 的页面，返回其余页面和隐藏的页数。
// 隐藏页在渲染前就被删除：页码、目录和首页封面都按剩下的页面计算，导出的 HTML 里也不含它们的内容。
func visible(pages []string) ([]string, int) {
	var kept []string
	for _, page := range pages {
		if !isHidden(page) {
			kept = append(kept, page)
		}
	}
	return kept, len(pages) - len(kept)
}

// isHidden 只检查页首连续的注释行，因此 hide 可以写在版式标记前后，正文里的同名注释不会误伤。
func isHidden(page string) bool {
	for _, line := range strings.Split(page, "\n") {
		switch {
		case hideRe.MatchString(line):
			return true
		case strings.TrimSpace(line) != "" && !commentLineRe.MatchString(line):
			return false
		}
	}
	return false
}

var (
	fenceRe      = regexp.MustCompile("^[ \t]*(`{3,}|~{3,})")
	fenceCloseRe = regexp.MustCompile("^[ \t]*(`{3,}|~{3,})[ \t]*$") // 结束围栏后面不能有信息字符串
	ruleRe       = regexp.MustCompile("^[ \t]*---[ \t]*$")
	colRe        = regexp.MustCompile("^[ \t]*::col::[ \t]*$")
	// 只认整行的 <!-- hide -->，不和版式合写，避免 <!-- TODO: hide --> 这类普通注释误隐藏页面
	hideRe        = regexp.MustCompile(`(?i)^[ \t]*<!--[ \t]*hide[ \t]*-->[ \t]*$`)
	commentLineRe = regexp.MustCompile(`^[ \t]*<!--.*-->[ \t]*$`)
	// \$ 也要保护：Markdown 会把它变成 $，MathJax 就会误认作公式边界；原样保留则由 MathJax 显示为 $
	mathRe = regexp.MustCompile(`\\\$|\$\$[\s\S]+?\$\$|\$(?:\\.|[^$\n])+\$`)
	h1Re   = regexp.MustCompile(`(?s)<h1[^>]*>(.*?)</h1\s*>`)
	tagRe  = regexp.MustCompile(`<[^>]*>`)

	footnoteRefRe = regexp.MustCompile(`<sup class="footnote-ref">\d+</sup>`)
)

func split(source string) []string {
	var pages, current []string
	var fenceChar byte
	fenceLen := 0
	flush := func() {
		if page := strings.TrimSpace(strings.Join(current, "\n")); page != "" {
			pages = append(pages, page)
		}
		current = nil
	}
	source = strings.ReplaceAll(source, "\r\n", "\n")
	for _, line := range strings.Split(source, "\n") {
		if fenceLen > 0 {
			if m := fenceCloseRe.FindStringSubmatch(line); m != nil && m[1][0] == fenceChar && len(m[1]) >= fenceLen {
				fenceLen = 0
			}
			current = append(current, line)
		} else if m := fenceRe.FindStringSubmatch(line); m != nil {
			fenceChar, fenceLen = m[1][0], len(m[1])
			current = append(current, line)
		} else if ruleRe.MatchString(line) {
			flush()
		} else if colRe.MatchString(line) {
			// 补上空行，否则紧跟在列表后的 ::col:: 会被并入最后一项；模板按这个注释分栏
			current = append(current, "", "<!-- col -->", "")
		} else {
			current = append(current, line)
		}
	}
	flush()
	return pages
}

// renderPage 先把 TeX 公式替换为占位符，避免 Markdown 改写其中的 _ * \ 等字符，渲染后再原样放回交给 MathJax。
func renderPage(md goldmark.Markdown, page string) string {
	var formulas []string
	protected := mathRe.ReplaceAllStringFunc(page, func(f string) string {
		formulas = append(formulas, f)
		return fmt.Sprintf("@@SLIDES_MATH_%d@@", len(formulas)-1)
	})
	var buf bytes.Buffer
	if err := md.Convert([]byte(protected), &buf); err != nil {
		return "<pre>" + html.EscapeString(err.Error()) + "</pre>"
	}
	out := strings.TrimSpace(buf.String())
	for i := len(formulas) - 1; i >= 0; i-- { // 倒序替换，避免 _1 误伤 _10
		escaped := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(formulas[i])
		out = strings.ReplaceAll(out, fmt.Sprintf("@@SLIDES_MATH_%d@@", i), escaped)
	}
	return out
}

func firstHeading(slide string) string {
	m := h1Re.FindStringSubmatch(slide)
	if m == nil {
		return ""
	}
	heading := footnoteRefRe.ReplaceAllString(m[1], "") // 否则脚注序号会混进页面标题
	return strings.TrimSpace(html.UnescapeString(tagRe.ReplaceAllString(heading, "")))
}

func newMarkdown() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.Table, extension.Strikethrough, extension.Footnote, markExtension{}),
		goldmark.WithParserOptions(
			parser.WithInlineParsers(util.Prioritized(extension.NewTaskCheckBoxParser(), 0)),
			parser.WithASTTransformers(util.Prioritized(taskListClass{}, 0)),
		),
		goldmark.WithRendererOptions(
			ghtml.WithUnsafe(), // 允许在 Markdown 中直接写 HTML
			renderer.WithNodeRenderers(util.Prioritized(taskCheckBoxRenderer{}, 0), util.Prioritized(footnoteRenderer{}, 0)),
		),
	)
}

// 任务列表输出与模板 CSS 对应的 class：li.task-list-item / input.task-list-item-checkbox。
type taskListClass struct{}

func (taskListClass) Transform(doc *gast.Document, reader text.Reader, pc parser.Context) {
	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if box, ok := n.(*extast.TaskCheckBox); ok && entering {
			if li := box.Parent().Parent(); li != nil && li.Kind() == gast.KindListItem {
				li.SetAttributeString("class", []byte("task-list-item"))
			}
		}
		return gast.WalkContinue, nil
	})
}

type taskCheckBoxRenderer struct{}

func (taskCheckBoxRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(extast.KindTaskCheckBox, func(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
		if entering {
			_, _ = w.WriteString(`<input class="task-list-item-checkbox" type="checkbox" disabled`)
			if n.(*extast.TaskCheckBox).IsChecked {
				_, _ = w.WriteString(" checked")
			}
			_, _ = w.WriteString("/>")
		}
		return gast.WalkContinue, nil
	})
}

// Options 控制 HTML 输出方式。
type Options struct {
	// Standalone 把运行库、字体和本地图片全部内联，得到可单独分发的一个 HTML 文件。
	// 否则保留相对路径（供预览服务器提供）。
	Standalone bool
	// Inject 在 </body> 前插入的额外 HTML，例如预览时的自动刷新脚本。
	Inject string
}

// HTML 把幻灯片套入模板。
func (d *Deck) HTML(opt Options) ([]byte, []string, error) {
	tpl := assets.Template()
	if b, err := os.ReadFile(filepath.Join(d.Dir, TemplateFile)); err == nil {
		tpl = string(b)
	}
	var warnings []string
	slides := d.Slides
	if opt.Standalone {
		slides = make([]string, len(d.Slides))
		for i, s := range d.Slides {
			slides[i] = imgSrcRe.ReplaceAllStringFunc(s, func(tag string) string {
				out, warn := inlineImage(d.Dir, tag)
				if warn != "" {
					warnings = append(warnings, fmt.Sprintf("第 %d 页：%s", i+1, warn))
				}
				return out
			})
		}
	}

	payload, err := json.Marshal(slides) // 默认会把 < > & U+2028 U+2029 转义，可安全放进 <script>
	if err != nil {
		return nil, nil, err
	}
	loc := dataRe.FindStringIndex(tpl)
	if loc == nil {
		return nil, nil, fmt.Errorf("模板中缺少 __SLIDES_DATA_START__ / __SLIDES_DATA_END__ 标记")
	}
	tpl = tpl[:loc[0]] + "// __SLIDES_DATA_START__\nconst MY_SLIDES = " + string(payload) + ";\n// __SLIDES_DATA_END__" + tpl[loc[1]:]

	if loc := titleRe.FindStringIndex(tpl); loc != nil {
		tpl = tpl[:loc[0]] + "<title>" + html.EscapeString(d.Title) + "</title>" + tpl[loc[1]:]
	}
	if opt.Standalone {
		tpl, err = inlineVendor(tpl)
		if err != nil {
			return nil, nil, err
		}
	}
	if opt.Inject != "" {
		if i := strings.LastIndex(tpl, "</body>"); i >= 0 {
			tpl = tpl[:i] + opt.Inject + "\n" + tpl[i:]
		} else {
			tpl += opt.Inject
		}
	}
	return []byte(tpl), warnings, nil
}

var (
	dataRe      = regexp.MustCompile(`// __SLIDES_DATA_START__\n[\s\S]*?// __SLIDES_DATA_END__`)
	titleRe     = regexp.MustCompile(`<title>[\s\S]*?</title>`)
	scriptSrcRe = regexp.MustCompile(`<script src="(vendor/[^"]+)"></script>`)
	cssURLRe    = regexp.MustCompile(`url\("(vendor/[^"]+)"\)`)
	imgSrcRe    = regexp.MustCompile(`<img\b[^>]*?\ssrc\s*=\s*("[^"]*"|'[^']*')`)
)

func inlineVendor(tpl string) (string, error) {
	var missing error
	tpl = scriptSrcRe.ReplaceAllStringFunc(tpl, func(tag string) string {
		name := scriptSrcRe.FindStringSubmatch(tag)[1]
		b, err := assets.FS.ReadFile(name)
		if err != nil {
			missing = err
			return tag
		}
		return "<script>" + string(b) + "</script>"
	})
	tpl = cssURLRe.ReplaceAllStringFunc(tpl, func(u string) string {
		name := cssURLRe.FindStringSubmatch(u)[1]
		b, err := assets.FS.ReadFile(name)
		if err != nil {
			missing = err
			return u
		}
		return `url("` + dataURI(name, b) + `")`
	})
	return tpl, missing
}

// inlineImage 把 <img src="相对路径"> 替换为 data URI；远程地址和 data URI 保持不变。
func inlineImage(dir, tag string) (string, string) {
	m := imgSrcRe.FindStringSubmatchIndex(tag)
	quoted := tag[m[2]:m[3]]
	src := html.UnescapeString(quoted[1 : len(quoted)-1])
	u, err := url.Parse(src)
	if err != nil || u.Scheme != "" || u.Host != "" || strings.HasPrefix(src, "/") || u.Path == "" {
		return tag, ""
	}
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(u.Path)))
	if err != nil {
		return tag, "找不到图片 " + src
	}
	return tag[:m[2]] + `"` + dataURI(u.Path, b) + `"` + tag[m[3]:], ""
}

func dataURI(name string, b []byte) string {
	typ := mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	switch strings.ToLower(filepath.Ext(name)) {
	case ".svg":
		typ = "image/svg+xml"
	case ".woff2":
		typ = "font/woff2"
	}
	if typ == "" {
		typ = "application/octet-stream"
	}
	if i := strings.Index(typ, ";"); i >= 0 {
		typ = typ[:i]
	}
	return "data:" + typ + ";base64," + base64.StdEncoding.EncodeToString(b)
}
