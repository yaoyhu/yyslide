package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaoyhu/yyslide/assets"
)

func TestSplitIgnoresRulesInsideFences(t *testing.T) {
	src := "# A\n\n````markdown\n---\n```\n---\n````\n\n---\n\n# B\n\n---\n\n"
	pages := split(src)
	if len(pages) != 2 {
		t.Fatalf("got %d pages: %q", len(pages), pages)
	}
	if !strings.Contains(pages[0], "---\n```\n---") {
		t.Errorf("fence content was split: %q", pages[0])
	}
	// ```python 这样带信息字符串的行不能结束代码块
	if pages := split("```\n```python\n---\n```\n\n---\n\nB"); len(pages) != 2 {
		t.Errorf("info string closed fence: %q", pages)
	}
}

func TestColumnSeparator(t *testing.T) {
	// 紧跟在列表后也要分栏；代码块里的写法示例保持原样
	got := Slides("- a\n::col::\n- b\n\n```\n::col::\n```\n")[0]
	if !strings.Contains(got, "</ul>\n<!-- col -->\n<ul>") {
		t.Errorf("separator not recognized:\n%s", got)
	}
	if !strings.Contains(got, "<code>::col::\n</code>") {
		t.Errorf("separator inside fence was rewritten:\n%s", got)
	}
}

func TestRenderPage(t *testing.T) {
	cases := []struct{ in, want string }{
		{"==警示==", "<mark>警示</mark>"},
		{"a === b", "a === b"},
		{"~~删~~", "<del>删</del>"},
		{"$a_1 < b_{2}$ 与 *x*", "$a_1 &lt; b_{2}$ 与 <em>x</em>"},
		{"$$\n\\frac{a*b}{c*d}\n$$", "$$\n\\frac{a*b}{c*d}\n$$"},
		{`价格 \$5 到 \$10，$x$`, `价格 \$5 到 \$10，$x$`},
		{"- [x] done\n- [ ] todo", `<li class="task-list-item"><input class="task-list-item-checkbox" type="checkbox" disabled checked/>done</li>`},
		{"<div class=\"x\">raw</div>", "<div class=\"x\">raw</div>"},
		{"```python\nprint(1)\n```", `<pre><code class="language-python">print(1)`},
		{"| a | b |\n| - | - |\n| 1 | 2 |", "<table>"},
		// 模板靠这个 class 找到 Mermaid 源码；$ 必须原样保留，不能被公式占位符破坏
		{"```mermaid\nflowchart LR\n  A[\"$5 到 $10\"] --> B\n```", "<pre><code class=\"language-mermaid\">flowchart LR\n  A[&quot;$5 到 $10&quot;] --&gt; B"},
	}
	md := newMarkdown()
	for _, c := range cases {
		if got := renderPage(md, c.in); !strings.Contains(got, c.want) {
			t.Errorf("render(%q)\n got: %s\nwant: %s", c.in, got, c.want)
		}
	}
}

func TestFootnotes(t *testing.T) {
	// 每页独立编号；定义可写在页内任意位置，公式不被 Markdown 改写，两处引用共用一个序号
	slides := Slides("# A[^x]\n\n[^x]: 来源 $a_1$\n\n再次引用[^x]\n\n---\n\n正文[^y]\n\n[^y]: 第二页\n")
	if len(slides) != 2 {
		t.Fatalf("got %d slides", len(slides))
	}
	for _, want := range []string{
		`<h1>A<sup class="footnote-ref">1</sup></h1>`,
		`再次引用<sup class="footnote-ref">1</sup>`,
		`<div class="footnotes"`,
		`来源 $a_1$`,
	} {
		if !strings.Contains(slides[0], want) {
			t.Errorf("slide 1 missing %q:\n%s", want, slides[0])
		}
	}
	if !strings.Contains(slides[1], `正文<sup class="footnote-ref">1</sup>`) {
		t.Errorf("numbering should restart on each slide:\n%s", slides[1])
	}
	// 锚点会改写记录页码的 #/N，id 则会在多页之间重复
	for i, s := range slides {
		if strings.Contains(s, "<a ") || strings.Contains(s, " id=") {
			t.Errorf("slide %d has anchors or ids:\n%s", i+1, s)
		}
	}
	if got := firstHeading(slides[0]); got != "A" {
		t.Errorf("title = %q", got)
	}

	// 定义在别页时无法解析，原样显示；没有引用的定义不输出脚注区
	if s := Slides("正文[^z]\n\n---\n\n[^z]: 别页的定义\n"); !strings.Contains(s[0], "[^z]") || strings.Contains(s[1], "footnotes") {
		t.Errorf("cross-page footnote: %q", s)
	}
}

func TestHTML(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "images"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "images", "a.svg"), []byte("<svg/>"), 0o644)
	md := filepath.Join(dir, "slides.md")
	_ = os.WriteFile(md, []byte("# 标题 <b>&</b>\n\n![](images/a.svg)\n\n---\n\n![](missing.png) ![](https://x.org/a.png)"), 0o644)

	deck, err := Load(md, "")
	if err != nil {
		t.Fatal(err)
	}
	if deck.Title != "标题 &" {
		t.Errorf("title = %q", deck.Title)
	}

	page, warnings, err := deck.HTML(Options{Standalone: true})
	if err != nil {
		t.Fatal(err)
	}
	s := string(page)
	for _, want := range []string{
		"<title>标题 &amp;</title>",
		"data:image/svg+xml;base64,",
		"https://x.org/a.png",
		`url("data:font/woff2;base64,`,
		"var pele=",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("standalone output missing %q", want)
		}
	}
	if strings.Contains(s, `src="vendor/`) || strings.Contains(s, "<b>&</b>") {
		t.Error("vendor not inlined or slide HTML not escaped in script")
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "missing.png") {
		t.Errorf("warnings = %q", warnings)
	}

	page, _, err = deck.HTML(Options{Inject: "<!--live-->"})
	if err != nil {
		t.Fatal(err)
	}
	if s := string(page); !strings.Contains(s, `<script src="vendor/mathjax-tex-svg.js">`) || !strings.Contains(s, "<!--live-->\n</body>") {
		t.Error("linked output should keep vendor paths and inject before </body>")
	}
}

// 独立 HTML 把运行库直接写进 <script>，其中出现 </script 会提前结束标签、破坏页面。
func TestVendorScriptsAreSafeToInline(t *testing.T) {
	entries, err := assets.FS.ReadDir("vendor")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		b, err := assets.FS.ReadFile("vendor/" + e.Name())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(string(b)), "</script") {
			t.Errorf("%s contains </script", e.Name())
		}
	}
}
