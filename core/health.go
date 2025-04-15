package core

import (
	"fmt"
	"net/http"
	"time"
)

func CheckHealth(port int) bool {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/actuator/health", port))
	if err != nil || resp.StatusCode != 200 {
		return false
	}
	return true
}

func HealthMonitor() {
	for {
		time.Sleep(30 * time.Second)
		if !CheckHealth(CurrentPort) {
			StopProcess(CurrentPort)
			StartProcess("./uploads/"+StandbyVersion, CurrentPort)
		}
	}
}
