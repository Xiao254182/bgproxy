package service

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// 版本列表
func versionsHandler(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()

	c.HTML(http.StatusOK, "versions.html", gin.H{
		"Versions": versions,
	})
}

// 备份版本
func backupVersion(instance *ServiceInstance) {
	backupPath := filepath.Join("bak", instance.Version+".jar")
	if err := os.Rename(instance.JarPath, backupPath); err != nil {
		log.Printf("❌ 备份失败: %v\n", err)
		return
	}
	versions = append(versions, VersionInfo{
		Time:    instance.Version,
		JarPath: backupPath,
	})
	log.Printf("📦 版本已备份: %s\n", backupPath)
}
