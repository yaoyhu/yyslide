package pdf

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chromedp/chromedp"

	"github.com/yaoyhu/yyslide/internal/render"
)

// standalone 生成独立 HTML。这些测试需要真实浏览器，本机没有时跳过。
func standalone(t *testing.T, markdown string) string {
	t.Helper()
	if _, err := FindBrowser(); err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	md := filepath.Join(dir, "slides.md")
	if err := os.WriteFile(md, []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}
	deck, err := render.Load(md, "")
	if err != nil {
		t.Fatal(err)
	}
	page, _, err := deck.HTML(render.Options{Standalone: true})
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "slides.html")
	if err := os.WriteFile(out, page, 0o644); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestExportReportsDiagramErrors(t *testing.T) {
	html := standalone(t, "# A\n\n---\n\n# B\n\n```mermaid\nflowchart LR\n  A --> B\n```\n\n---\n\n# C\n\n```mermaid\nflowchart LR\n  A --> --> [[[\n```\n")
	out := filepath.Join(t.TempDir(), "out.pdf")
	warnings, err := Export(html, out)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || !strings.HasPrefix(warnings[0], "第 3 页：") || !strings.Contains(warnings[0], "Parse error") {
		t.Errorf("warnings = %q", warnings)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Error("output is not a PDF")
	}
}

func TestExportWithoutDiagramsHasNoWarnings(t *testing.T) {
	html := standalone(t, "# A\n\n正文\n")
	warnings, err := Export(html, filepath.Join(t.TempDir(), "out.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %q", warnings)
	}
}

// MathJax 会扫描全文：不隔离的话，标签里的 "$5 到 $" 会被当成公式吃掉。
func TestDiagramLabelsSurviveMathJax(t *testing.T) {
	html := standalone(t, "# A\n\n$E=mc^2$\n\n```mermaid\nflowchart LR\n  A[\"价格 $5 到 $10\"] --> B\n```\n")
	ctx, cancel, err := openPrintView(html)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	text, err := chromedp.Run(ctx, chromedp.Evaluate[string](`document.querySelector('.diagram').textContent`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "价格 $5 到 $10") {
		t.Errorf("diagram text = %q", text)
	}
	typeset, err := chromedp.Run(ctx, chromedp.Evaluate[int](`document.querySelectorAll('.inner > p mjx-container').length`))
	if err != nil {
		t.Fatal(err)
	}
	if typeset != 1 {
		t.Errorf("body formula typeset %d times, want 1", typeset)
	}
}

func TestLayouts(t *testing.T) {
	html := standalone(t, "# 首页默认封面\n\n---\n\n<!-- section -->\n# 节\n\n---\n\n# 普通\n\n---\n\n<!-- cover -->\n# 封面\n\n---\n\n"+
		"<!-- image-right crop -->\n# 图\n\n文字\n\n![](a.png)\n\n---\n\n<!-- image-right -->\n# 没有图片\n\n文字\n")
	ctx, cancel, err := openPrintView(html)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	classes, err := chromedp.Run(ctx, chromedp.Evaluate[string](`[...document.querySelectorAll('.slide')].map(s =>
		[...s.classList].filter(c => !['slide', 'active', 'no-anim', 'stage'].includes(c)).join(' ')).join('|')`))
	if err != nil {
		t.Fatal(err)
	}
	if want := "cover|section||cover|image-right crop|"; classes != want {
		t.Errorf("layouts = %q, want %q", classes, want)
	}
	// 图片移到右栏后，左栏只剩文字，不留空段落
	left, err := chromedp.Run(ctx, chromedp.Evaluate[string](`(() => {
		const s = document.querySelector('.slide.image-right');
		return s.querySelectorAll('.pane img').length + ':' + s.querySelector('.inner').innerHTML.trim();
	})()`))
	if err != nil {
		t.Fatal(err)
	}
	if want := "1:<p>文字</p>"; left != want {
		t.Errorf("image-right = %q, want %q", left, want)
	}
}

func TestLayoutStructure(t *testing.T) {
	html := standalone(t, "# 首页\n\n---\n\n<!-- section -->\n# 第一章[^1]\n\n[^1]: 注\n\n---\n\n<!-- agenda -->\n# 目录\n\n---\n\n"+
		"<!-- section -->\n# 第二章\n\n---\n\n<!-- two-cols -->\n# 两栏\n\n- 左\n::col::\n- 右\n\n---\n\n<!-- three-cols -->\n# 三栏\n\n甲\n\n::col::\n\n乙\n\n::col::\n\n丙\n\n---\n\n"+
		"<!-- figure -->\n# 结论\n\n![图注](a.png)\n\n---\n\n<!-- image-left -->\n# 左图\n\n文字\n\n![](a.png)\n\n---\n\n<!-- full -->\n\n![](a.png)\n")
	ctx, cancel, err := openPrintView(html)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	cases := []struct{ name, js, want string }{
		{"two-cols", `[...document.querySelectorAll('.two-cols .col')].map(c => c.textContent.trim()).join('|')`, "左|右"},
		{"three-cols", `[...document.querySelectorAll('.three-cols .col')].map(c => c.textContent.trim()).join('|')`, "甲|乙|丙"},
		{"figure", `document.querySelector('.figure figure').innerHTML.replace(/ src="[^"]*"/, '')`, `<img alt="图注"><figcaption>图注</figcaption>`},
		{"figure no empty paragraph", `String(document.querySelectorAll('.figure .inner p').length)`, "0"},
		{"image-left", `document.querySelectorAll('.image-left .pane img').length + ':' + document.querySelector('.image-left .inner').textContent.trim()`, "1:文字"},
		{"full without text", `document.querySelectorAll('.full .pane img').length + ':' + document.querySelector('.full .inner').classList.contains('blank')`, "1:true"},
		// 目录排除脚注序号，并高亮本页所属的章节
		{"agenda", `[...document.querySelectorAll('.agenda-list li')].map(l => (l.className || '-') + ':' + l.textContent + ':' + l.querySelector('a').getAttribute('href')).join('|')`, "now:第一章:#/2|-:第二章:#/4"},
	}
	for _, c := range cases {
		got, err := chromedp.Run(ctx, chromedp.Evaluate[string](c.js))
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
		} else if got != c.want {
			t.Errorf("%s = %q, want %q", c.name, got, c.want)
		}
	}
}

// accTitle 生成的 <title id> 和 sankey 的渐变都带 id，多张图之间不能重名，否则引用会指向别的图。
func TestDiagramIDsAreUnique(t *testing.T) {
	src := "```mermaid\nflowchart LR\n  accTitle: 标题\n  A --> B\n```\n\n"
	html := standalone(t, "# A\n\n"+src+src)
	ctx, cancel, err := openPrintView(html)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	dup, err := chromedp.Run(ctx, chromedp.Evaluate[int](`(() => {
		const ids = [...document.querySelectorAll('.diagram [id]')].map(e => e.id);
		return ids.length === 0 ? -1 : ids.length - new Set(ids).size;
	})()`))
	if err != nil {
		t.Fatal(err)
	}
	if dup != 0 {
		t.Errorf("duplicate ids = %d (-1 means no ids found)", dup)
	}
}
