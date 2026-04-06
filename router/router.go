package router

import (
	"github.com/gin-gonic/gin"
	"goworkers/handler"
	"goworkers/middleware"
)

func Setup(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/register", handler.Register)
		api.POST("/login", handler.Login)

		auth := api.Group("")
		auth.Use(middleware.AuthMiddleware())
		{
			auth.GET("/profile", handler.GetProfile)
			auth.PUT("/profile", handler.UpdateProfile)
		}
	}
}
