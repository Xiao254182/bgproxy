package utils

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// 版本回滚
func rollbackHandler(c *gin.Context) {
	versionTime := c.PostForm("version")

	// 查找对应版本
	var targetVersion *VersionInfo
	for _, v := range versions {
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

	if err := startNewService(targetVersion.JarPath, port); err != nil {
		c.String(http.StatusInternalServerError, "启动失败: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/")
}
