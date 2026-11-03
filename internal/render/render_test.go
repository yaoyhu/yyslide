package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	}
	md := newMarkdown()
	for _, c := range cases {
		if got := renderPage(md, c.in); !strings.Contains(got, c.want) {
			t.Errorf("render(%q)\n got: %s\nwant: %s", c.in, got, c.want)
		}
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
