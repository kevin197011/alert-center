package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		tokenString := parts[1]
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid user_id in token"})
			return
		}
		c.Set("user_id", userID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Role not found"})
			return
		}

		userRole := role.(string)
		for _, allowedRole := range allowedRoles {
			if userRole == allowedRole {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	}
}

const (
	RoleAdmin  = "admin"
	RoleManager = "manager"
	RoleUser   = "user"
)

var rolePermissions = map[string][]string{
	RoleAdmin: {
		"users:read", "users:write", "users:delete",
		"rules:read", "rules:write", "rules:delete",
		"channels:read", "channels:write", "channels:delete",
		"templates:read", "templates:write", "templates:delete",
		"silences:read", "silences:write", "silences:delete",
		"data-sources:read", "data-sources:write", "data-sources:delete",
		"audit-logs:read",
		"statistics:read",
	},
	RoleManager: {
		"rules:read", "rules:write", "rules:delete",
		"channels:read", "channels:write", "channels:delete",
		"templates:read", "templates:write", "templates:delete",
		"silences:read", "silences:write", "silences:delete",
		"data-sources:read", "data-sources:write",
		"statistics:read",
	},
	RoleUser: {
		"rules:read",
		"channels:read",
		"templates:read",
		"silences:read",
		"data-sources:read",
	},
}

func PermissionMiddleware(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Role not found"})
			return
		}

		userRole := role.(string)
		permissions, ok := rolePermissions[userRole]
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Unknown role"})
			return
		}

		for _, p := range permissions {
			if p == permission {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
	}
}

func HasPermission(role string, permission string) bool {
	permissions, ok := rolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}
