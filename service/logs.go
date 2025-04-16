package service

import (
	"bufio"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"time"
)

// 日志接口
func streamLogHandler(c *gin.Context) {
	service := c.Param("service")
	full := c.DefaultQuery("full", "0") == "1" // 获取 full 参数
	var instance *ServiceInstance

	mu.Lock()
	if service == "active" {
		instance = activeInstance
	} else if service == "new" {
		instance = newInstance
	}
	mu.Unlock()

	if instance == nil {
		c.String(http.StatusNotFound, "服务实例不存在")
		return
	}

	logFile := fmt.Sprintf("./logs/%s.log", instance.Version)
	file, err := os.Open(logFile)
	if err != nil {
		c.String(http.StatusInternalServerError, "无法打开日志文件: %v", err)
		return
	}
	defer file.Close()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	reader := bufio.NewReader(file)

	// 根据 full 参数决定是否读取历史日志
	if full {
		file.Seek(0, 0) // 文件指针返回到文件开头
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", line)
			c.Writer.Flush()
		}
	}

	// 跳到文件末尾，开始实时读取新增日志
	file.Seek(0, 2) // 直接跳到文件末尾

	// 实时读取新增日志
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		fmt.Fprintf(c.Writer, "data: %s\n\n", line)
		c.Writer.Flush()
	}
}
