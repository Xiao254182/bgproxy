package proxy

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// 切换服务
func switchHandler(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()

	if newInstance == nil || newInstance.Status != StatusRunning {
		c.String(http.StatusBadRequest, "新服务未就绪")
		return
	}

	// 停止旧服务
	if activeInstance != nil {
		stopService(activeInstance)
	}

	// 更新代理
	updateProxy(newInstance.Port)

	// 备份旧版本
	if activeInstance != nil {
		backupVersion(activeInstance)
	}

	// 切换实例
	activeInstance = newInstance
	newInstance = nil

	c.Redirect(http.StatusFound, "/")
}

// 反向代理处理
func reverseProxyHandler(c *gin.Context) {
	proxy.ServeHTTP(c.Writer, c.Request)
}

// 更新反向代理
func updateProxy(port int) {
	target, _ := url.Parse(fmt.Sprintf("http://localhost:%d", port))
	proxy = httputil.NewSingleHostReverseProxy(target)
	log.Printf("🔁 代理更新至端口: %d\n", port)
}
