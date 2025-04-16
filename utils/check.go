package utils

import (
	"fmt"
	"net"
)

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
