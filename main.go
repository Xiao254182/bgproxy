package main

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// 全局配置
const (
	JarPath      = "/usr/share/service/app.jar"
	UploadDir    = "/usr/share/service/versions"
	ReadTimeout  = 30 * time.Second
	WriteTimeout = 30 * time.Second
)

// 全局状态
var (
	CurrentPort      = 8081
	StandbyPort      = 8082
	currentVersion   = "v1.0.0"
	standbyVersion   = "v1.0.0"
	activeProcess    *os.Process
	mu               sync.RWMutex
	requestCounter   *prometheus.CounterVec
	requestHistogram *prometheus.HistogramVec
)

// 初始化Prometheus指标
func initMetrics() {
	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	requestHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration",
			Buckets: []float64{0.1, 0.3, 0.5, 1, 2},
		},
		[]string{"method", "path", "status"},
	)

	prometheus.MustRegister(requestCounter, requestHistogram)
}

// 主函数
func main() {
	initMetrics()

	// 初始化Gin引擎
	router := gin.Default()
	router.SetFuncMap(template.FuncMap{
		"formatTime": formatTime,
	})
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./static")

	// 中间件
	router.Use(authMiddleware())
	router.Use(metricsMiddleware())

	// 路由配置
	router.GET("/", dashboardHandler)
	router.GET("/status", statusHandler)
	router.GET("/versions", listVersionsHandler)
	router.POST("/upload", uploadHandler)
	router.POST("/deploy", deployHandler)
	router.POST("/rollback", rollbackHandler)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 启动后台任务
	go healthMonitor()

	// 启动HTTP服务
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
	}

	// 优雅关闭
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server shutdown error:", err)
	}
}

// 仪表盘处理器
func dashboardHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.tmpl", gin.H{
		"CurrentVersion": currentVersion,
		"StandbyVersion": standbyVersion,
		"HealthStatus":   checkHealth(CurrentPort),
	})
}

// 添加 listVersionsHandler 空实现
func listVersionsHandler(c *gin.Context) {
	files, err := filepath.Glob(filepath.Join(UploadDir, "app-*.jar"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	versions := []string{}
	for _, file := range files {
		versions = append(versions, filepath.Base(file))
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

// 添加 rollbackHandler 空实现
func rollbackHandler(c *gin.Context) {
	// 这里简单地交换版本（模拟回滚）
	mu.Lock()
	currentVersion, standbyVersion = standbyVersion, currentVersion
	CurrentPort, StandbyPort = StandbyPort, CurrentPort
	mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"message": "Rollback triggered",
	})
}

// 状态接口
func statusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"current":  currentVersion,
		"standby":  standbyVersion,
		"healthy":  checkHealth(CurrentPort),
		"requests": getRequestCount(),
	})
}

// 文件上传处理器
func uploadHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	version := c.PostForm("version")
	if version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version required"})
		return
	}

	dst := filepath.Join(UploadDir, fmt.Sprintf("app-%s.jar", version))
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Uploaded %s", version),
		"path":    dst,
	})
}

// 部署处理器
func deployHandler(c *gin.Context) {
	version := c.PostForm("version")
	if version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version required"})
		return
	}

	if err := performDeployment(version); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Deployed %s", version),
	})
}

// 执行部署
func performDeployment(version string) error {
	// 停止备用进程
	if err := stopProcess(standbyVersion); err != nil {
		return err
	}

	// 启动新版本
	if err := startProcess(version, StandbyPort); err != nil {
		return err
	}

	// 健康检查
	if !checkHealth(StandbyPort) {
		return fmt.Errorf("health check failed for %s", version)
	}

	// 切换流量
	mu.Lock()
	currentVersion, standbyVersion = standbyVersion, currentVersion
	CurrentPort, StandbyPort = StandbyPort, CurrentPort
	mu.Unlock()

	// 清理旧版本
	go func() {
		time.Sleep(5 * time.Minute)
		os.Remove(filepath.Join(UploadDir, fmt.Sprintf("app-%s.jar", standbyVersion)))
	}()

	return nil
}

// 启动Java进程
func startProcess(version string, port int) error {
	cmd := exec.Command("java", "-jar",
		filepath.Join(UploadDir, fmt.Sprintf("app-%s.jar", version)),
		fmt.Sprintf("--server.port=%d", port),
	)

	logFile, err := os.Create(fmt.Sprintf("/var/log/app/%s.log", version))
	if err != nil {
		return err
	}

	cmd.Stdout = io.MultiWriter(os.Stdout, logFile)
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return err
	}

	mu.Lock()
	activeProcess = cmd.Process
	mu.Unlock()

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("Process %s exited: %v", version, err)
		}
	}()

	return nil
}

// 停止进程
func stopProcess(version string) error {
	mu.Lock()
	defer mu.Unlock()

	if activeProcess != nil {
		if err := activeProcess.Kill(); err != nil {
			return err
		}
	}
	return nil
}

// 健康检查
func checkHealth(port int) bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/health", port))
	return err == nil && resp.StatusCode == http.StatusOK
}

// 认证中间件
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-API-Key") != os.Getenv("API_KEY") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		c.Next()
	}
}

// 指标中间件
func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		status := fmt.Sprintf("%d", c.Writer.Status())

		requestCounter.WithLabelValues(
			c.Request.Method,
			c.Request.URL.Path,
			status,
		).Inc()

		requestHistogram.WithLabelValues(
			c.Request.Method,
			c.Request.URL.Path,
			status,
		).Observe(duration.Seconds())
	}
}

// 其他辅助函数
func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func getRequestCount() float64 {
	if counter, err := requestCounter.GetMetricWithLabelValues("GET", "/", "200"); err == nil {
		val := testutil.ToFloat64(counter)
		return val
	}
	return 0
}

func healthMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		if !checkHealth(CurrentPort) {
			log.Printf("Critical: current version (%s) is unhealthy", currentVersion)
		}
	}
}
