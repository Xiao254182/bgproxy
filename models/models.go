package models

import (
	"net/http/httputil"
	"os/exec"
	"sync"
	"time"
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
	ActiveInstance *ServiceInstance
	NewInstance    *ServiceInstance
	Versions       []VersionInfo
	Mu             sync.Mutex
	Proxy          *httputil.ReverseProxy
)
