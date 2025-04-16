package router

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// 前端页面
func indexHandler(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Active": activeInstance,
		"New":    newInstance,
	})
}

func Router() *gin.Engine {
	r := gin.Default()

	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"safeStatus":  safeStatus,
		"safeTime":    safeTime,
		"safeVersion": safeVersion,
	}).ParseGlob("templates/*"))
	r.SetHTMLTemplate(tmpl)

	r.Static("/static", "./static")

	r.GET("/", indexHandler)
	r.GET("/status", statusHandler)
	r.GET("/versions", versionsHandler)
	r.POST("/upload", uploadHandler)
	r.POST("/switch", switchHandler)
	r.POST("/rollback", rollbackHandler)
	r.Any("/service/*path", reverseProxyHandler)
	r.GET("/stream-log/:service", streamLogHandler)
}
