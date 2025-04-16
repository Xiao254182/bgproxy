package service

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
	"time"
)

// 服务状态接口
func statusHandler(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"active": map[string]interface{}{
			"status":    safeStatus(activeInstance),
			"startTime": safeTime(activeInstance),
			"version":   safeVersion(activeInstance),
		},
		"new": map[string]interface{}{
			"status":    safeStatus(newInstance),
			"startTime": safeTime(newInstance),
			"version":   safeVersion(newInstance),
		},
	})
}

// 服务监控
func monitorService(instance *ServiceInstance) {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if checkHealth(instance.Port) {
				mu.Lock()
				instance.Status = StatusRunning
				mu.Unlock()
				return
			}
		case <-timeout:
			mu.Lock()
			instance.Status = StatusError
			mu.Unlock()
			return
		}
	}
}

// 健康检查
func checkHealth(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
