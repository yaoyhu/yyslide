package render

import (
	"strconv"

	gast "github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// 脚注的语法和解析沿用 goldmark 的 footnote 扩展，这里只替换输出：去掉所有锚点。
// 脚注与引用在同一屏，无需跳转；默认的 #fn:1 每页重复，点击还会覆盖记录页码的 #/N。
// 外层 <div class="footnotes"> 沿用默认输出，模板靠它把脚注移到页底。

type footnoteRenderer struct{}

func (footnoteRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(extast.KindFootnoteLink, func(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
		if entering {
			_, _ = w.WriteString(`<sup class="footnote-ref">` + strconv.Itoa(n.(*extast.FootnoteLink).Index) + `</sup>`)
		}
		return gast.WalkContinue, nil
	})
	reg.Register(extast.KindFootnoteBacklink, func(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
		return gast.WalkContinue, nil
	})
	reg.Register(extast.KindFootnote, func(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
		if entering {
			_, _ = w.WriteString("<li>\n")
		} else {
			_, _ = w.WriteString("</li>\n")
		}
		return gast.WalkContinue, nil
	})
}
