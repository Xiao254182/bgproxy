package service

import (
	"bgproxy/models"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// 版本列表
func VersionsHandler(c *gin.Context) {
	models.Mu.Lock()
	defer models.Mu.Unlock()

	c.HTML(http.StatusOK, "versions.html", gin.H{
		"Versions": models.Versions,
	})
}

// 备份版本
func BackupVersion(instance *models.ServiceInstance) {
	backupPath := filepath.Join("bak", instance.Version+".jar")
	if err := os.Rename(instance.JarPath, backupPath); err != nil {
		log.Printf("❌ 备份失败: %v\n", err)
		return
	}
	models.Versions = append(models.Versions, models.VersionInfo{
		Time:    instance.Version,
		JarPath: backupPath,
	})
	log.Printf("📦 版本已备份: %s\n", backupPath)
}
