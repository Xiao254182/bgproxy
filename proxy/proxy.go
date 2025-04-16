package proxy

import (
	"bgproxy/models"
	"bgproxy/service"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// 切换服务
func SwitchHandler(c *gin.Context) {
	models.Mu.Lock()
	defer models.Mu.Unlock()

	if models.NewInstance == nil || models.NewInstance.Status != models.StatusRunning {
		c.String(http.StatusBadRequest, "新服务未就绪")
		return
	}

	// 停止旧服务
	if models.ActiveInstance != nil {
		service.StopService(models.ActiveInstance)
	}

	// 更新代理
	UpdateProxy(models.NewInstance.Port)

	// 备份旧版本
	if models.ActiveInstance != nil {
		service.BackupVersion(models.ActiveInstance)
	}

	// 切换实例
	models.ActiveInstance = models.NewInstance
	models.NewInstance = nil

	c.Redirect(http.StatusFound, "/")
}

// 反向代理处理
func ReverseProxyHandler(c *gin.Context) {
	models.Proxy.ServeHTTP(c.Writer, c.Request)
}

// 更新反向代理
func UpdateProxy(port int) {
	target, _ := url.Parse(fmt.Sprintf("http://localhost:%d", port))
	models.Proxy = httputil.NewSingleHostReverseProxy(target)
	log.Printf("🔁 代理更新至端口: %d\n", port)
}
