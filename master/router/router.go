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

			auth.POST("/tasks", handler.CreateTask)
			auth.GET("/tasks", handler.GetTasks)
			auth.GET("/tasks/:id", handler.GetTask)
			auth.PUT("/tasks/:id", handler.UpdateTask)
			auth.DELETE("/tasks/:id", handler.DeleteTask)
			auth.GET("/tasks/:id/logs", handler.GetTaskLogs)
		}

		// Worker 接口（无需用户认证）
		workers := api.Group("/workers")
		{
			workers.POST("", handler.RegisterWorker)
			workers.POST("/:id/heartbeat", handler.Heartbeat)
			workers.GET("/:id/next-task", handler.GetNextTask)
			workers.POST("/:id/logs", handler.SubmitLog)
			workers.POST("/:id/tasks/:task_id/complete", handler.CompleteTask)
			workers.POST("/:id/tasks/:task_id/fail", handler.FailTask)
		}
	}
}
