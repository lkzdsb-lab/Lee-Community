package middleware

import (
	"net/http"
	"strings"

	"Lee_Community/internal/pkg"
	"Lee_Community/internal/repository/redis"

	"github.com/gin-gonic/gin"
)

const ContextUserIDKey = "user_id"

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"msg": "missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"msg": "invalid authorization format"})
			c.Abort()
			return
		}

		tokenStr := parts[1]
		userRep := &redis.UserRepository{}

		claims, err := pkg.ParseAccess(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"msg": "invalid or expired token"})
			c.Abort()
			return
		}

		// redis校验是否是正确的token
		OriginToken, err := userRep.GetUserToken(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"msg": "Account has been logging elsewhere"})
			c.Abort()
			return
		}

		if OriginToken != tokenStr {
			c.JSON(http.StatusUnauthorized, gin.H{"msg": "Account has been logging elsewhere"})
			c.Abort()
			return
		}

		// 校验通过后更新过期时间
		err = userRep.ExtendUserToken(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
			return
		}

		// 注入 user_id
		c.Set(ContextUserIDKey, claims.UserID)
		c.Next()
	}
}
