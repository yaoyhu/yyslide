// Package preview 提供本地预览服务器：Markdown、图片或模板变化后浏览器自动刷新。
package preview

import (
	"fmt"
	"html"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/yaoyhu/yyslide/assets"
	"github.com/yaoyhu/yyslide/internal/render"
)

const eventsPath = "/_yyslide/events"

// 自动刷新：保留地址中的 #/N，刷新后停在当前页
const reloadScript = `<script>(() => {
  const es = new EventSource('` + eventsPath + `');
  es.addEventListener('reload', () => location.reload());
})();</script>`

// Serve 启动预览服务器并阻塞运行。ready 在开始监听后以访问地址回调一次。
func Serve(mdPath, title, addr string, ready func(url string)) error {
	abs, err := filepath.Abs(mdPath)
	if err != nil {
		return err
	}
	dir := filepath.Dir(abs)
	hub := newHub()

	vendor, _ := fs.Sub(assets.FS, "vendor")
	mux := http.NewServeMux()
	mux.Handle("/vendor/", http.StripPrefix("/vendor/", http.FileServer(http.FS(vendor))))
	mux.HandleFunc(eventsPath, hub.serve)
	files := http.FileServer(http.Dir(dir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			files.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		page, err := renderPage(abs, title)
		if err != nil {
			page = []byte(errorPage(err))
		}
		_, _ = w.Write(page)
	})

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	if err := watch(dir, abs, title, hub); err != nil {
		return err
	}
	if ready != nil {
		ready("http://" + displayAddr(ln.Addr().String()))
	}
	return http.Serve(ln, mux)
}

func renderPage(path, title string) ([]byte, error) {
	deck, err := render.Load(path, title)
	if err != nil {
		return nil, err
	}
	page, _, err := deck.HTML(render.Options{Inject: reloadScript})
	return page, err
}

func errorPage(err error) string {
	return `<!DOCTYPE html><meta charset="utf-8"><title>yyslide 错误</title>
<body style="font: 16px/1.6 -apple-system, sans-serif; padding: 2em; color: #b91c1c">
<h2>无法渲染幻灯片</h2><pre style="white-space: pre-wrap">` + html.EscapeString(err.Error()) + `</pre>` + reloadScript
}

func displayAddr(a string) string {
	host, port, err := net.SplitHostPort(a)
	if err != nil {
		return a
	}
	if host == "" || host == "::" || host == "0.0.0.0" || host == "127.0.0.1" || host == "::1" {
		host = "localhost"
	}
	return net.JoinHostPort(host, port)
}

// watch 递归监听 deck 目录（跳过 dist/ 和隐藏目录），合并短时间内的连续事件后通知浏览器刷新。
func watch(dir, mdPath, title string, hub *hub) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	skip := func(path string) bool {
		rel, err := filepath.Rel(dir, path)
		if err != nil || rel == "." {
			return false
		}
		first := strings.Split(filepath.ToSlash(rel), "/")[0]
		return first == "dist" || first == "node_modules" || strings.HasPrefix(first, ".")
	}
	add := func(root string) {
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			if skip(p) {
				return filepath.SkipDir
			}
			_ = w.Add(p)
			return nil
		})
	}
	add(dir)

	go func() {
		var timer *time.Timer
		fire := func() {
			if _, err := renderPage(mdPath, title); err != nil {
				log.Printf("渲染失败：%v", err)
			} else {
				log.Printf("已更新")
			}
			hub.broadcast()
		}
		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if skip(ev.Name) || ev.Op == fsnotify.Chmod {
					continue
				}
				if ev.Op&fsnotify.Create != 0 {
					if st, err := os.Stat(ev.Name); err == nil && st.IsDir() {
						add(ev.Name)
					}
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(120*time.Millisecond, fire)
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				log.Printf("监听文件出错：%v", err)
			}
		}
	}()
	return nil
}

// hub 管理所有打开的预览页面（Server-Sent Events 连接）。
type hub struct {
	mu      sync.Mutex
	clients map[chan struct{}]struct{}
}

func newHub() *hub { return &hub{clients: map[chan struct{}]struct{}{}} }

func (h *hub) broadcast() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c <- struct{}{}:
		default:
		}
	}
}

func (h *hub) serve(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	c := make(chan struct{}, 1)
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.clients, c)
		h.mu.Unlock()
	}()
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()
	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-c:
			fmt.Fprint(w, "event: reload\ndata: \n\n")
			flusher.Flush()
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
