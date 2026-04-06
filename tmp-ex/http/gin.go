package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var database = make(map[int]User)
var idindex int = 1

func createUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindBodyWithJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Println(user.Name)
	fmt.Println(user.Email)
	user.ID = idindex
	idindex++
	fmt.Println(user.ID)
	database[user.ID] = user

	c.JSON(http.StatusCreated, user)
}

func getUsers(c *gin.Context) {
	users := make([]User, 0, len(database))
	for _, user := range database {
		fmt.Printf("user id: %d, the name is: %s, email add: %s \n", user.ID, user.Name, user.Email)
		users = append(users, user)
	}
	c.JSON(http.StatusOK, users)
}

func getUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}

	user, ok := database[id]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func deleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}

	if _, ok := database[id]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	delete(database, id)
	c.Status(http.StatusNoContent)
}

func main() {
	r := gin.Default()
	r.POST("/users", createUser)
	r.GET("/users", getUsers)
	r.GET("/users/:id", getUser)
	r.DELETE("/users/:id", deleteUser)

	r.Run(":8080")
}
