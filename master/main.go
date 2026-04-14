package main

import (
	"goworkers/config"
	"goworkers/model"
	"goworkers/router"
	"goworkers/service"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	config.InitDB()
	config.InitRedis()
	config.DB.AutoMigrate(&model.User{})
	config.DB.AutoMigrate(&model.Worker{})
	config.DB.AutoMigrate(&model.TaskLog{})
	config.DB.AutoMigrate(&model.Task{})

	// 后台定期将 Redis 心跳状态同步回 DB
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			service.SyncWorkerStatus()
		}
	}()

	r := gin.Default()
	router.Setup(r)
	r.Run(":8080")
}
