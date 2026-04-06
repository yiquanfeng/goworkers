package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no token"})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(tokenString, "Bearer ")
		token_parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return []byte("spriple-jwt-key"), nil
		})

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "decrypt failed"})
			c.Abort()
			return
		}

		claims := token_parsed.Claims.(jwt.MapClaims)
		userId := uint(claims["user_id"].(float64))
		c.Set("user_id", userId)

		c.Next()
	}
}
