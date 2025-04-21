package main

import (
	"bgproxy/proxy"
	"bgproxy/router"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"os"
)

// 将web端模板和静态文件打包进可执行文件
//
//go:embed templates/*.html
var templatesFS embed.FS

//go:embed templates/static/*
var staticFS embed.FS

func main() {
	// 模板解析
	tmpl := template.Must(template.ParseFS(templatesFS, "templates/*.html"))

	// 静态资源子目录
	staticContent, err := fs.Sub(staticFS, "templates/static")
	if err != nil {
		panic(err)
	}

	// 创建需要的目录
	_ = os.MkdirAll("bak", 0755)
	_ = os.MkdirAll("uploads", 0755)
	_ = os.MkdirAll("logs", 0755)

	// 初始化代理配置
	proxy.UpdateProxy(8080)

	// 启动 Gin 路由
	r := router.Router(tmpl, staticContent)

	log.Println("🔥 管理平台已启动: http://localhost:3000")
	log.Fatal(r.Run(":3000"))
}
