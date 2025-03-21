package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	mu               sync.RWMutex
	activeBackendURL *url.URL
	currentCmd       *exec.Cmd // 保存当前正在运行的jar进程句柄
)

func setActiveBackend(u *url.URL) {
	mu.Lock()
	defer mu.Unlock()
	activeBackendURL = u
}

func getActiveBackend() *url.URL {
	mu.RLock()
	defer mu.RUnlock()
	return activeBackendURL
}

// tcpProbe 检查指定地址的TCP连接是否可用
func tcpProbe(address string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// reverseProxyHandler 将所有流量转发到当前激活的jar服务
func reverseProxyHandler(w http.ResponseWriter, r *http.Request) {
	backend := getActiveBackend()
	if backend == nil {
		http.Error(w, "backend not set", http.StatusServiceUnavailable)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(backend)
	proxy.ServeHTTP(w, r)
}

// startJarProcess 启动jar包进程，jarPath指定jar文件路径，port为启动端口
func startJarProcess(jarPath string, port int) (*exec.Cmd, error) {
	cmd := exec.Command("java", "-jar", jarPath, fmt.Sprintf("--server.port=%d", port))
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// autoUpdate 封装自动更新逻辑：启动新jar、健康检测、切换流量并终止旧进程
func autoUpdate(newJarPath string) {
	newPort := 8081
	log.Printf("自动更新：启动新jar进程，jar路径：%s, 端口：%d", newJarPath, newPort)
	newCmd, err := startJarProcess(newJarPath, newPort)
	if err != nil {
		log.Printf("启动新jar失败: %v", err)
		return
	}
	// 等待新jar启动并通过TCP探测
	healthAddr := fmt.Sprintf("localhost:%d", newPort)
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		if tcpProbe(healthAddr, 5*time.Second) {
			log.Println("新jar服务TCP探测成功")
			break
		}
		log.Println("等待新jar服务TCP探测...")
		time.Sleep(5 * time.Second)
	}
	if !tcpProbe(healthAddr, 5*time.Second) {
		log.Println("新jar服务在规定时间内未通过TCP探测，终止新jar进程")
		newCmd.Process.Kill()
		return
	}

	// 更新反向代理，切换流量到新jar服务
	newURL, _ := url.Parse(fmt.Sprintf("http://localhost:%d", newPort))
	setActiveBackend(newURL)
	log.Println("流量切换到新jar服务")
	// 杀死旧jar进程
	if currentCmd != nil {
		err := currentCmd.Process.Kill()
		if err != nil {
			log.Printf("终止旧jar进程失败: %v", err)
		} else {
			log.Println("旧jar进程已终止")
		}
	}
	// 更新全局变量保存新jar进程
	currentCmd = newCmd
}

// watchJarFile 监控指定目录下的jar文件，当目标jar文件被修改时触发自动更新
func watchJarFile(dir, targetFile string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("创建文件监控失败: %v", err)
	}
	// 添加目录监控
	err = watcher.Add(dir)
	if err != nil {
		log.Fatalf("添加监控目录失败: %v", err)
	}
	log.Printf("开始监控目录: %s，目标文件: %s", dir, targetFile)
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// 过滤出目标文件的事件
			if filepath.Clean(event.Name) == filepath.Clean(targetFile) &&
				(event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				log.Printf("检测到目标jar文件变化: %v", event)
				// 自动触发更新操作
				autoUpdate(targetFile)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("文件监控错误: %v", err)
		}
	}
}

func main() {
	// 启动旧jar进程（监听8080），旧jar路径保持不变
	oldPort := 8080
	oldJarPath := "/opt/yjzh/server/old.jar"
	cmd, err := startJarProcess(oldJarPath, oldPort)
	if err != nil {
		log.Fatalf("启动旧jar失败: %v", err)
	}
	currentCmd = cmd
	backendURL, _ := url.Parse(fmt.Sprintf("http://localhost:%d", oldPort))
	setActiveBackend(backendURL)
	log.Printf("旧jar进程启动在端口 %d", oldPort)

	// 监控所在目录（例如 /usr/share/service），目标文件为旧jar（被替换时触发更新）
	dirToWatch := "/opt/yjzh/server"
	// 这里目标文件路径与容器内实际的jar路径保持一致
	targetFile := "/opt/yjzh/server/old.jar"
	go watchJarFile(dirToWatch, targetFile)

	http.HandleFunc("/", reverseProxyHandler)

	log.Println("管理服务启动在端口 80")
	log.Fatal(http.ListenAndServe(":80", nil))
}
