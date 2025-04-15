package main

import (
	"bytes"
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
	r.GET("/log", logHandler)

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
func logHandler(c *gin.Context) {
	if newInstance == nil {
		c.String(http.StatusNotFound, "没有可用实例")
		return
	}
	c.String(http.StatusOK, newInstance.Cmd.Stdout.(*bytes.Buffer).String())
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
	instance := &ServiceInstance{}

	if err := startNewService(instance, newJar, port); err != nil {
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

	instance := &ServiceInstance{}

	if err := startNewService(instance, targetVersion.JarPath, port); err != nil {
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
func startNewService(instance *ServiceInstance, jarPath string, port int) error {
	mu.Lock()
	defer mu.Unlock()

	log.Printf("🟡 启动新服务：%s，端口：%d\n", jarPath, port)

	cmd := exec.Command("java", "-jar", jarPath, "--server.port="+strconv.Itoa(port))
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Start(); err != nil {
		log.Printf("❌ 启动失败：%v\n", err)
		return err
	}

	// // 启动后台协程等待结束，避免僵尸进程
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Printf("🛑 停止旧服务子进程:（%d）：%v\n", cmd.Process.Pid, err)
		}
	}()

	newInstance = &ServiceInstance{
		Port:      port,
		PID:       cmd.Process.Pid,
		Status:    StatusStarting,
		StartTime: time.Now(),
		JarPath:   jarPath,
		Version:   time.Now().Format("2025-04-15_15-04-05"),
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
