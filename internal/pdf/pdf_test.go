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
