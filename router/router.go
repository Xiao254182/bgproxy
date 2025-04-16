package router

import (
	"bgproxy/models"
	"bgproxy/proxy"
	"bgproxy/service"
	"bgproxy/utils"
	"github.com/gin-gonic/gin"
	"html/template"
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

func Router() *gin.Engine {
	r := gin.Default()

	tmpl := template.Must(template.New("").ParseGlob("templates/*.html"))
	r.SetHTMLTemplate(tmpl)
	r.Static("/static", "./templates/static")

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
