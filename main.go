package main

import (
	"bgproxy/core"
	"bgproxy/web"
)

func main() {
	go core.HealthMonitor()
	web.StartServer()
}
