package router

import (
	"bgproxy/models"
	"bgproxy/proxy"
	"bgproxy/service"
	"bgproxy/utils"
	"github.com/gin-gonic/gin"
	"html/template"
	"io/fs"
	"net/http"
)

// 前端页面
func indexHandler(c *gin.Context) {
	models.Mu.Lock()
	defer models.Mu.Unlock()

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Active": models.ActiveInstance,
		"New":    models.NewInstance,
	})
}

// Router 创建 Gin 引擎，接收预处理好的模板和静态文件系统
func Router(tmpl *template.Template, staticFS fs.FS) *gin.Engine {
	r := gin.Default()

	// 设置模板和静态资源
	r.SetHTMLTemplate(tmpl)
	r.StaticFS("/static", http.FS(staticFS))

	// 注册路由
	r.GET("/", indexHandler)
	r.GET("/status", service.StatusHandler)
	r.GET("/versions", service.VersionsHandler)
	r.POST("/upload", utils.UploadHandler)
	r.POST("/switch", proxy.SwitchHandler)
	r.POST("/rollback", utils.RollbackHandler)
	r.Any("/service/*path", proxy.ReverseProxyHandler)
	r.GET("/stream-log/:service", service.StreamLogHandler)

	return r
}
