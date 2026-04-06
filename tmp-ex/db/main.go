package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

type User struct {
	gorm.Model
	Name  string `json:"name"`
	Email string `json:"email"`
}

func initdb() {
	dsn := "host=localhost user=postgres password=123456 dbname=ginpractice port=5432 sslmode=disable"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&User{})

}

func createUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.Create(&user)
	c.JSON(http.StatusCreated, user)
}

func getUsers(c *gin.Context) {
	var users []User
	db.Find(&users)
	c.JSON(http.StatusOK, users)
}

func getUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	// panic("test")
	var user User
	if result := db.First(&user, id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)

}

func deleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "the bad param"})
		return
	}

	var user User
	if result := db.First(&user, id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	db.Delete(&user, id)
	c.Status(http.StatusNoContent)
}

func updateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var old User
	var new User
	if result := db.First(&old, id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := c.ShouldBindJSON(&new); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	old.Name = new.Name
	old.Email = new.Email
	db.Save(&old)
	c.JSON(http.StatusOK, new)
}

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		fmt.Println("start logger")
		c.Next()
		fmt.Printf("the method is %s, the path is %s, the status code is %d, consumed %d tick", c.Request.Method, c.FullPath(), c.Writer.Status(), time.Since(start))
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "Bearer my-secret-token" {
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not allowed to do"})
			c.Abort()
		}
	}
}

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("recover, the error is")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				c.Abort()
			}
		}()
		c.Next()
	}
}

func main() {
	initdb()
	r := gin.New()
	r.Use(LoggerMiddleware())
	r.Use(RecoveryMiddleware())

	users := r.Group("/users")
	users.Use(AuthMiddleware())
	{
		users.POST("", createUser)
		users.GET("", getUsers)
		users.GET("/:id", getUser)
		users.DELETE("/:id", deleteUser)
		users.PUT("/:id", updateUser)
	}

	r.Run(":8080")
}
