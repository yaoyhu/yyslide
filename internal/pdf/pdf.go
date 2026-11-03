// Package pdf 借助本机 Chrome / Chromium / Edge 的无界面模式把幻灯片打印为 PDF。
//
// 幻灯片的排版（CSS、自动缩放）、代码高亮和公式都在浏览器里由 JavaScript 完成，
// 只有真正的浏览器引擎才能得到与播放时完全一致的效果，因此这里驱动浏览器“打印”。
package pdf

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// 与模板中 @page { size: 1280px 720px } 一致
const width, height = 1280, 720

// Export 把一个独立 HTML 文件（已内联全部资源）打印为 PDF。
func Export(htmlPath, pdfPath string) error {
	chrome, err := FindBrowser()
	if err != nil {
		return err
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chrome),
		chromedp.WindowSize(width, height),
		chromedp.Flag("allow-file-access-from-files", true),
	)
	if runtime.GOOS == "linux" && os.Geteuid() == 0 {
		opts = append(opts, chromedp.NoSandbox) // Docker / CI 中以 root 运行时 Chrome 拒绝启用沙箱
	}
	actx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	ctx, cancel := chromedp.NewContext(actx)
	defer cancel()
	abs, err := filepath.Abs(htmlPath)
	if err != nil {
		return err
	}
	// Windows 路径 C:\x 需要写成 file:///C:/x
	target := (&url.URL{Scheme: "file", Path: "/" + strings.TrimPrefix(filepath.ToSlash(abs), "/"), RawQuery: "print"}).String()

	if err := chromedp.Do(ctx,
		chromedp.EmulateViewport(width, height),
		chromedp.Navigate(target),
	); err != nil {
		return fmt.Errorf("浏览器打开页面失败：%w", err)
	}
	// 等待模板在公式、字体、图片就绪并完成排版后设置 data-ready
	if _, err := chromedp.Run(ctx, chromedp.Poll[chromedp.Void](
		`document.documentElement.dataset.ready === "1"`,
		chromedp.WithPollingInterval(100*time.Millisecond),
		chromedp.WithPollingTimeout(2*time.Minute),
	)); err != nil {
		return fmt.Errorf("等待页面渲染完成失败（公式或图片加载过慢？）：%w", err)
	}
	buf, err := chromedp.Run(ctx, chromedp.PrintToPDF(
		chromedp.PDFPrintBackground(),
		chromedp.PDFPreferCSSPageSize(),
		chromedp.PDFMargin(0),
	))
	if err != nil {
		return fmt.Errorf("浏览器导出失败：%w", err)
	}
	return os.WriteFile(pdfPath, buf, 0o644)
}

// FindBrowser 查找可用的 Chromium 内核浏览器；可用环境变量 YYSLIDE_CHROME 指定路径。
func FindBrowser() (string, error) {
	if p := os.Getenv("YYSLIDE_CHROME"); p != "" {
		return p, nil
	}
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		for _, app := range []string{
			"Google Chrome.app/Contents/MacOS/Google Chrome",
			"Chromium.app/Contents/MacOS/Chromium",
			"Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"Brave Browser.app/Contents/MacOS/Brave Browser",
		} {
			candidates = append(candidates, "/Applications/"+app, filepath.Join(os.Getenv("HOME"), "Applications", app))
		}
	case "windows":
		for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LocalAppData"} {
			base := os.Getenv(env)
			if base == "" {
				continue
			}
			candidates = append(candidates,
				filepath.Join(base, `Google\Chrome\Application\chrome.exe`),
				filepath.Join(base, `Microsoft\Edge\Application\msedge.exe`),
				filepath.Join(base, `Chromium\Application\chrome.exe`))
		}
	default:
		for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge", "brave-browser"} {
			if p, err := exec.LookPath(name); err == nil {
				candidates = append(candidates, p)
			}
		}
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", errors.New("没有找到 Chrome / Chromium / Edge。导出 PDF 需要其中之一，安装后重试；也可以用环境变量 YYSLIDE_CHROME 指定浏览器路径")
}
