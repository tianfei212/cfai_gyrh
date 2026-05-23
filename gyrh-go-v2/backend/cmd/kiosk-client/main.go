package main

import (
	"context"
	"flag"
	"log"

	"gyrh-go-v2/backend/internal/kiosk"
)

func main() {
	configPath := flag.String("config", kiosk.DefaultConfigPath, "kiosk client config file")
	flag.Parse()

	if err := kiosk.Run(context.Background(), *configPath); err != nil {
		log.Fatalf("kiosk 客户端启动失败: %v", err)
	}
}
