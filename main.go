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

	r := gin.Default()
	router.Setup(r)
	r.Run(":8080")
}
