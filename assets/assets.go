// Package assets 打包幻灯片模板、浏览器端运行库和 `yyslide new` 的示例工程。
package assets

import "embed"

//go:embed template.html vendor starter
var FS embed.FS

// Template 是默认的幻灯片页面模板。
func Template() string {
	b, err := FS.ReadFile("template.html")
	if err != nil {
		panic(err)
	}
	return string(b)
}
