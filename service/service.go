package service

import (
	"bgproxy/models"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

// 启动新服务
func StartNewService(jarPath string, port int) error {
	models.Mu.Lock()
	defer models.Mu.Unlock()

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
	models.NewInstance = &models.ServiceInstance{
		Port:      port,
		PID:       cmd.Process.Pid,
		Status:    models.StatusStarting,
		StartTime: time.Now(),
		JarPath:   jarPath,
		Version:   version, // 使用时间戳作为唯一版本号
	}

	go monitorService(models.NewInstance)
	return nil
}

// 停止服务
func StopService(instance *models.ServiceInstance) error {
	log.Printf("🛑 停止旧服务 PID: %d\n", instance.PID)
	err := syscall.Kill(instance.PID, syscall.SIGKILL)
	if err != nil {
		return fmt.Errorf("无法杀死进程 %d: %w", instance.PID, err)
	}
	return nil
}
