package middlewares

import (
	"net/http"
	"strings"

	"go-demo/services"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware JWT Bearer Token 认证中间件
func AuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" { c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header is required"}); return }
		parts := strings.Fields(authHeader)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") { c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header must use Bearer token"}); return }
		claims, err := authService.ValidateAccessToken(parts[1])
		if err != nil { c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "access token is invalid or expired"}); return }
		user, err := authService.GetUserByID(c.Request.Context(), claims.UserID)
		if err != nil || user.DeletedAt != nil { c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "user is deactivated"}); return }
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Next()
	}
}

// RequirePermission 基于权限的访问控制中间件
func RequirePermission(authService *services.AuthService, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDValue, exists := c.Get("user_id")
		if !exists { c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"}); return }
		userID, ok := userIDValue.(int64)
		if !ok { c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"}); return }
		allowed, err := authService.UserHasPermission(c.Request.Context(), userID, permission)
		if err != nil { c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to check permission"}); return }
		if !allowed { c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"}); return }
		c.Next()
	}
}
