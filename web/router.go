package web

import (
	"bgproxy/core"
	"bgproxy/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

func StartServer() {
	router := gin.Default()
	router.Use(middleware.MetricsMiddleware())
	router.Use(middleware.AuthMiddleware("your-api-key"))

	router.Static("/static", "./static")
	router.LoadHTMLGlob("templates/*")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard.tmpl", gin.H{
			"version": core.CurrentVersion,
			"port":    core.CurrentPort,
		})
	})

	router.POST("/deploy", func(c *gin.Context) {
		version := c.PostForm("version")
		if err := core.PerformDeployment(version); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "deployed"})
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	router.Run(":8082")
}
