package main

import (
	"context"
	"log"

	"gyrh-go-v2/backend/internal/platform/app"
)

// main 是后端服务的最小启动入口，实际依赖装配由 platform/app 负责。
func main() {
	if err := app.Run(context.Background()); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
