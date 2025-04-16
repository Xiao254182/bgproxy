package utils

import (
	"bgproxy/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"path/filepath"
)

// 文件上传
func UploadHandler(c *gin.Context) {
	file, err := c.FormFile("jar")
	if err != nil {
		c.String(http.StatusBadRequest, "上传失败: "+err.Error())
		return
	}

	// 保存新文件
	newJar := filepath.Join("uploads", file.Filename)
	if err := c.SaveUploadedFile(file, newJar); err != nil {
		c.String(http.StatusInternalServerError, "保存失败: "+err.Error())
		return
	}

	// 启动新实例
	port := findAvailablePort()
	if port == 0 {
		c.String(http.StatusInternalServerError, "没有可用端口")
		return
	}
	if err := service.StartNewService(newJar, port); err != nil {
		c.String(http.StatusInternalServerError, "启动失败: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/")
}
