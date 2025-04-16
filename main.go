package main

import (
	"bufio"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type ServiceStatus string

const (
	StatusRunning  ServiceStatus = "running"
	StatusStarting               = "starting"
	StatusError                  = "error"
)

type ServiceInstance struct {
	Port      int
	PID       int
	Cmd       *exec.Cmd // 新增：记录启动的进程
	Status    ServiceStatus
	StartTime time.Time
	JarPath   string
	Version   string
}

type VersionInfo struct {
	Time    string
	JarPath string
}

var (
	activeInstance *ServiceInstance
	newInstance    *ServiceInstance
	versions       []VersionInfo
	mu             sync.Mutex
	proxy          *httputil.ReverseProxy
)

func init() {
	_ = os.MkdirAll("bak", 0755)
	_ = os.MkdirAll("uploads", 0755)
	updateProxy(8080)
}

func main() {
	r := gin.Default()

	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"safeStatus":  safeStatus,
		"safeTime":    safeTime,
		"safeVersion": safeVersion,
	}).ParseGlob("templates/*"))
	r.SetHTMLTemplate(tmpl)

	r.Static("/static", "./static")

	r.GET("/", indexHandler)
	r.GET("/status", statusHandler)
	r.GET("/versions", versionsHandler)
	r.POST("/upload", uploadHandler)
	r.POST("/switch", switchHandler)
	r.POST("/rollback", rollbackHandler)
	r.Any("/service/*path", reverseProxyHandler)
	r.GET("/stream-log/:service", streamLogHandler)

	log.Println("🔥 管理平台已启动: http://localhost:3000")
	log.Fatal(r.Run(":3000"))
}

// 前端页面
func indexHandler(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Active": activeInstance,
		"New":    newInstance,
	})
}

// 日志接口
func streamLogHandler(c *gin.Context) {
	service := c.Param("service")
	full := c.DefaultQuery("full", "0") == "1" // 新增：获取 full 参数
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
		file.Seek(0, 0)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", line)
			c.Writer.Flush()
		}
	} else {
		file.Seek(0, 2) // 直接跳到文件末尾
	}

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

// 版本列表
func versionsHandler(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()

	c.HTML(http.StatusOK, "versions.html", gin.H{
		"Versions": versions,
	})
}

// 文件上传
func uploadHandler(c *gin.Context) {
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
	if err := startNewService(newJar, port); err != nil {
		c.String(http.StatusInternalServerError, "启动失败: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/")
}

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

// 反向代理处理
func reverseProxyHandler(c *gin.Context) {
	proxy.ServeHTTP(c.Writer, c.Request)
}

// 启动新服务
func startNewService(jarPath string, port int) error {
	mu.Lock()
	defer mu.Unlock()

	log.Printf("🟡 启动新服务：%s，端口：%d\n", jarPath, port)

	version := time.Now().Format("2006-01-02_15-04-05")
	logFilePath := fmt.Sprintf("./logs/%s.log", version)

	// 以追加模式打开文件，避免覆盖之前的日志
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("❌ 创建日志文件失败: %v\n", err)
		return err
	}

	cmd := exec.Command("java", "-jar", jarPath, "--server.port="+strconv.Itoa(port))
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		log.Printf("❌ 启动失败：%v\n", err)
		return err
	}

	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Printf("🛑 服务进程异常退出:（%d）：%v\n", cmd.Process.Pid, err)
		}
	}()

	// 设置新实例
	newInstance = &ServiceInstance{
		Port:      port,
		PID:       cmd.Process.Pid,
		Status:    StatusStarting,
		StartTime: time.Now(),
		JarPath:   jarPath,
		Version:   version, // 使用时间戳作为唯一版本号
	}

	go monitorService(newInstance)
	return nil
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

// 停止服务
func stopService(instance *ServiceInstance) error {
	log.Printf("🛑 停止旧服务 PID: %d\n", instance.PID)
	err := syscall.Kill(instance.PID, syscall.SIGKILL)
	if err != nil {
		return fmt.Errorf("无法杀死进程 %d: %w", instance.PID, err)
	}
	return nil
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

// 更新反向代理
func updateProxy(port int) {
	target, _ := url.Parse(fmt.Sprintf("http://localhost:%d", port))
	proxy = httputil.NewSingleHostReverseProxy(target)
	log.Printf("🔁 代理更新至端口: %d\n", port)
}

// 其他工具函数
func findAvailablePort() int {
	for port := 8080; port < 9000; port++ {
		if checkPortAvailable(port) {
			return port
		}
	}
	return 0
}

func checkPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

func safeStatus(instance *ServiceInstance) ServiceStatus {
	if instance == nil {
		return StatusError
	}
	return instance.Status
}

func safeTime(instance *ServiceInstance) string {
	if instance == nil {
		return ""
	}
	return instance.StartTime.Format("2025-04-15_15-04-05")
}

func safeVersion(instance *ServiceInstance) string {
	if instance == nil {
		return ""
	}
	return instance.Version
}
