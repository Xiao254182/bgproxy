package utils

import (
	"bgproxy/models"
	"bgproxy/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// 版本回滚
func RollbackHandler(c *gin.Context) {
	versionTime := c.PostForm("version")

	// 查找对应版本
	var targetVersion *models.VersionInfo
	for _, v := range models.Versions {
		if v.Time == versionTime {
			targetVersion = &v
			break
		}
	}

	if targetVersion == nil {
		c.String(http.StatusNotFound, "版本不存在")
		return
	}

	// 启动新实例
	port := findAvailablePort()
	if port == 0 {
		c.String(http.StatusInternalServerError, "没有可用端口")
		return
	}

	if err := service.StartNewService(targetVersion.JarPath, port); err != nil {
		c.String(http.StatusInternalServerError, "启动失败: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/")
}
