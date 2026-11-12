// yyslide：用 Markdown 写科大蓝风格幻灯片，一键预览、构建 HTML、导出 PDF。
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/yaoyhu/yyslide/assets"
	"github.com/yaoyhu/yyslide/internal/pdf"
	"github.com/yaoyhu/yyslide/internal/preview"
	"github.com/yaoyhu/yyslide/internal/render"
)

// 发布时通过 -ldflags "-X main.version=..." 注入
var version = ""

const usage = `yyslide — 用 Markdown 写幻灯片

用法：
  yyslide new <目录>          新建幻灯片工程（含示例 slides.md）
  yyslide preview [文件.md]   本地实时预览，修改后浏览器自动刷新
  yyslide build [文件.md]     生成单文件 HTML（资源全部内联，可离线播放、直接分享）
  yyslide pdf [文件.md]       导出 PDF（需要本机安装 Chrome / Edge / Chromium）
  yyslide eject [目录]        复制默认模板为 template.html，用于自定义样式
  yyslide version             显示版本

不指定 Markdown 文件时，使用当前目录的 slides.md。
运行 yyslide <命令> -h 查看各命令的选项。
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "new", "init":
		err = cmdNew(args)
	case "preview", "serve", "dev":
		err = cmdPreview(args)
	case "build", "html":
		err = cmdBuild(args)
	case "pdf":
		err = cmdPDF(args)
	case "eject":
		err = cmdEject(args)
	case "version", "-v", "--version":
		fmt.Println("yyslide", getVersion())
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "未知命令：%s\n\n%s", cmd, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误：%v\n", err)
		os.Exit(1)
	}
}

func getVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "dev"
}

// parse 解析选项，允许选项与位置参数混排（如 yyslide pdf talk.md -o x.pdf）。
func parse(fset *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fset.Parse(args); err != nil {
			return nil, err
		}
		args = fset.Args()
		if len(args) == 0 {
			return positional, nil
		}
		positional = append(positional, args[0])
		args = args[1:]
	}
}

// resolveInput 找到要处理的 Markdown：显式参数（文件或目录）> ./slides.md > 当前目录唯一的 .md。
func resolveInput(args []string) (string, error) {
	if len(args) > 1 {
		return "", errors.New("只能指定一个 Markdown 文件")
	}
	target := "."
	if len(args) == 1 {
		target = args[0]
	}
	st, err := os.Stat(target)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return target, nil
	}
	if p := filepath.Join(target, "slides.md"); fileExists(p) {
		return p, nil
	}
	matches, _ := filepath.Glob(filepath.Join(target, "*.md"))
	var mds []string
	for _, m := range matches {
		if !strings.EqualFold(filepath.Base(m), "README.md") {
			mds = append(mds, m)
		}
	}
	if len(mds) == 1 {
		return mds[0], nil
	}
	if len(mds) == 0 {
		return "", fmt.Errorf("%s 中没有 slides.md；请指定 Markdown 文件，或先运行 yyslide new <目录>", target)
	}
	return "", fmt.Errorf("%s 中有多个 Markdown 文件，请指定其中一个", target)
}

// defaultOutput：<deck>/dist/<名称>.<ext>；文件名为 slides.md 时以目录名命名。
func defaultOutput(md, ext string) string {
	abs, _ := filepath.Abs(md)
	dir := filepath.Dir(abs)
	name := strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
	if name == "slides" {
		name = filepath.Base(dir)
	}
	return filepath.Join(dir, "dist", name+ext)
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func cmdNew(args []string) error {
	fset := flag.NewFlagSet("new", flag.ExitOnError)
	fset.Usage = func() { fmt.Fprintln(os.Stderr, "用法：yyslide new <目录>") }
	pos, err := parse(fset, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		fset.Usage()
		os.Exit(2)
	}
	dir := pos[0]
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return fmt.Errorf("目录 %s 已存在且不为空", dir)
	}
	err = fs.WalkDir(assets.FS, "starter", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(p, "starter")))
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		b, err := assets.FS.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o644)
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("dist/\n.DS_Store\n"), 0o644); err != nil {
		return err
	}
	fmt.Printf("已创建 %s/\n\n  cd %s\n  yyslide preview   # 边写边看\n  yyslide pdf       # 导出 PDF\n", dir, dir)
	return nil
}

func cmdBuild(args []string) error {
	fset := flag.NewFlagSet("build", flag.ExitOnError)
	out := fset.String("o", "", "输出文件（默认 dist/<名称>.html）")
	title := fset.String("title", "", "页面标题（默认使用首页标题）")
	fset.Usage = func() {
		fmt.Fprintln(os.Stderr, "用法：yyslide build [文件.md] [-o 输出.html] [-title 标题]")
		fset.PrintDefaults()
	}
	pos, err := parse(fset, args)
	if err != nil {
		return err
	}
	md, err := resolveInput(pos)
	if err != nil {
		return err
	}
	if *out == "" {
		*out = defaultOutput(md, ".html")
	}
	deck, err := writeStandalone(md, *title, *out)
	if err != nil {
		return err
	}
	fmt.Printf("已生成 %s：%s\n", pageCount(deck), rel(*out))
	return nil
}

// pageCount 同时报出隐藏页数，免得忘了某页还标着 <!-- hide -->。
func pageCount(deck *render.Deck) string {
	if deck.Hidden > 0 {
		return fmt.Sprintf("%d 页（隐藏 %d 页）", len(deck.Slides), deck.Hidden)
	}
	return fmt.Sprintf("%d 页", len(deck.Slides))
}

func writeStandalone(md, title, out string) (*render.Deck, error) {
	deck, err := render.Load(md, title)
	if err != nil {
		return nil, err
	}
	page, warnings, err := deck.HTML(render.Options{Standalone: true})
	if err != nil {
		return nil, err
	}
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "警告：%s\n", w)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return nil, err
	}
	return deck, os.WriteFile(out, page, 0o644)
}

func cmdPDF(args []string) error {
	fset := flag.NewFlagSet("pdf", flag.ExitOnError)
	out := fset.String("o", "", "输出文件（默认 dist/<名称>.pdf）")
	title := fset.String("title", "", "文档标题（默认使用首页标题）")
	fset.Usage = func() {
		fmt.Fprintln(os.Stderr, "用法：yyslide pdf [文件.md] [-o 输出.pdf]\n\n浏览器路径可用环境变量 YYSLIDE_CHROME 指定。")
		fset.PrintDefaults()
	}
	pos, err := parse(fset, args)
	if err != nil {
		return err
	}
	md, err := resolveInput(pos)
	if err != nil {
		return err
	}
	if *out == "" {
		*out = defaultOutput(md, ".pdf")
	}
	tmp, err := os.MkdirTemp("", "yyslide-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	htmlPath := filepath.Join(tmp, "slides.html")
	deck, err := writeStandalone(md, *title, htmlPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	warnings, err := pdf.Export(htmlPath, *out)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "警告：%s\n", w)
	}
	fmt.Printf("已导出 %s：%s\n", pageCount(deck), rel(*out))
	return nil
}

func cmdPreview(args []string) error {
	fset := flag.NewFlagSet("preview", flag.ExitOnError)
	port := fset.Int("p", 8090, "端口（被占用时自动换一个）")
	host := fset.String("host", "127.0.0.1", "监听地址（0.0.0.0 允许局域网访问）")
	noOpen := fset.Bool("no-open", false, "不自动打开浏览器")
	title := fset.String("title", "", "页面标题")
	fset.Usage = func() {
		fmt.Fprintln(os.Stderr, "用法：yyslide preview [文件.md] [-p 端口] [-no-open]")
		fset.PrintDefaults()
	}
	pos, err := parse(fset, args)
	if err != nil {
		return err
	}
	md, err := resolveInput(pos)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(*host, strconv.Itoa(*port))
	if ln, err := net.Listen("tcp", addr); err != nil {
		addr = net.JoinHostPort(*host, "0")
	} else {
		ln.Close()
	}
	return preview.Serve(md, *title, addr, func(url string) {
		fmt.Printf("预览 %s：%s\n修改 Markdown 或图片后页面自动刷新，Ctrl+C 退出。\n", rel(md), url)
		if !*noOpen {
			openBrowser(url)
		}
	})
}

func cmdEject(args []string) error {
	fset := flag.NewFlagSet("eject", flag.ExitOnError)
	force := fset.Bool("f", false, "覆盖已有的 template.html")
	pos, err := parse(fset, args)
	if err != nil {
		return err
	}
	dir := "."
	if len(pos) > 0 {
		dir = pos[0]
	}
	dst := filepath.Join(dir, render.TemplateFile)
	if fileExists(dst) && !*force {
		return fmt.Errorf("%s 已存在（使用 -f 覆盖）", dst)
	}
	if err := os.WriteFile(dst, []byte(assets.Template()), 0o644); err != nil {
		return err
	}
	fmt.Printf("已复制模板到 %s，之后的 preview / build / pdf 都会使用它。\n", rel(dst))
	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func rel(p string) string {
	wd, err := os.Getwd()
	if err != nil {
		return p
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	if r, err := filepath.Rel(wd, abs); err == nil && !strings.HasPrefix(r, "..") {
		return r
	}
	return abs
}
