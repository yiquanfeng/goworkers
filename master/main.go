package main

import (
	"goworkers/config"
	"goworkers/model"
	"goworkers/router"

	"github.com/gin-gonic/gin"
)

func main() {
	config.InitDB()
	config.DB.AutoMigrate(&model.User{})
	config.DB.AutoMigrate(&model.Worker{})
	config.DB.AutoMigrate(&model.TaskLog{})
	config.DB.AutoMigrate(&model.Task{})

	r := gin.Default()
	router.Setup(r)
	r.Run(":8080")
}
