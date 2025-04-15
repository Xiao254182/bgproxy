package core

import (
	"fmt"
	"os/exec"
)

var (
	CurrentVersion string
	StandbyVersion string
	CurrentPort    = 8080
	StandbyPort    = 8081
)

func StartProcess(jarPath string, port int) error {
	cmd := exec.Command("java", "-jar", jarPath, "--server.port="+fmt.Sprint(port))
	return cmd.Start()
}

func StopProcess(port int) {
	exec.Command("fuser", "-k", fmt.Sprint(port)+"/tcp").Run()
}

func PerformDeployment(version string) error {
	jarPath := "./uploads/" + version
	StopProcess(StandbyPort)
	if err := StartProcess(jarPath, StandbyPort); err != nil {
		return err
	}
	// 假设 standby 健康，就切换 current/standby
	CurrentVersion, StandbyVersion = version, CurrentVersion
	CurrentPort, StandbyPort = StandbyPort, CurrentPort
	return nil
}
