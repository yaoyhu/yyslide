package render

import (
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// ==标记== → <mark>标记</mark>，行为与 strikethrough 扩展一致，只是分隔符为两个等号。

var kindMark = gast.NewNodeKind("Mark")

type markNode struct{ gast.BaseInline }

func (n *markNode) Kind() gast.NodeKind           { return kindMark }
func (n *markNode) Dump(source []byte, level int) { gast.DumpHelper(n, source, level, nil, nil) }

type markDelimiter struct{}

func (markDelimiter) IsDelimiter(b byte) bool { return b == '=' }
func (markDelimiter) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}
func (markDelimiter) OnMatch(consumes int) gast.Node { return &markNode{} }

type markParser struct{}

func (markParser) Trigger() []byte { return []byte{'='} }

func (markParser) Parse(parent gast.Node, block text.Reader, pc parser.Context) gast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 2, markDelimiter{})
	if node == nil || node.OriginalLength != 2 || before == '=' {
		return nil
	}
	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

func (markParser) CloseBlock(parent gast.Node, pc parser.Context) {}

type markRenderer struct{}

func (markRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindMark, func(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
		if entering {
			_, _ = w.WriteString("<mark>")
		} else {
			_, _ = w.WriteString("</mark>")
		}
		return gast.WalkContinue, nil
	})
}

type markExtension struct{}

func (markExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(util.Prioritized(markParser{}, 500)))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(markRenderer{}, 500)))
}
