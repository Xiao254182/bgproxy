package main

import (
	"log"
	"os"
)

func main() {
	r := setupRouter()

	_ = os.MkdirAll("bak", 0755)
	_ = os.MkdirAll("uploads", 0755)
	_ = os.MkdirAll("logs", 0755)

	updateProxy(8080)

	log.Println("🔥 管理平台已启动: http://localhost:3000")
	log.Fatal(r.Run(":3000"))
}
