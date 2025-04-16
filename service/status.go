package service

import (
	"bgproxy/models"
	"fmt"
	"github.com/gin-gonic/gin"
	"net"
	"time"
)

// 服务状态接口
func StatusHandler(c *gin.Context) {
	models.Mu.Lock()
	defer models.Mu.Unlock()
}

// 服务监控
func monitorService(instance *models.ServiceInstance) {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if checkHealth(instance.Port) {
				models.Mu.Lock()
				instance.Status = models.StatusRunning
				models.Mu.Unlock()
				return
			}
		case <-timeout:
			models.Mu.Lock()
			instance.Status = models.StatusError
			models.Mu.Unlock()
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
