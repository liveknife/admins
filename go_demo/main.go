package main

import (
	"fmt"
	"log"

	"go-demo/config"
	"go-demo/database"
	"go-demo/routes"
	"go-demo/services"
)

func main() {
	config.Init()

	port := config.Port()
	config.ValidateProductionConfig()

	db, err := database.Connect()
	if err != nil { log.Fatalf("failed to connect database: %v", err) }
	defer db.Close()

	if redisClient, err := database.ConnectRedis(); err != nil {
		fmt.Printf("[REDIS] warning: redis not available (%v)\n", err)
	} else { defer redisClient.Close() }

	r := routes.Setup(db.DB)

	services.InitMailer() // 初始化邮件服务（未配置 SMTP 时自动降级为 dry-run 模式）

	log.Printf("[%s] server running at http://localhost:%s", config.GetAppEnv(), port)
	if config.IsProduction() { log.Print("[INFO] database=", database.CurrentDialect.Type, " | redis=", database.RedisClient != nil, " | gin=release") } else { config.PrintDevConfig() }

	if err := r.Run(":" + port); err != nil { log.Fatalf("failed to start server: %v", err) }
}
