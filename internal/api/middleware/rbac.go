package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
)

type RBACMiddleware struct {
	permissions map[string][]string
}

func NewRBACMiddleware(cfg config.RBACConfig) *RBACMiddleware {
	permissions := make(map[string][]string)
	for role, roleCfg := range cfg.Roles {
		permissions[role] = roleCfg.Permissions
	}
	return &RBACMiddleware{
		permissions: permissions,
	}
}

func (m *RBACMiddleware) CheckPermission(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Role not found in context"})
			return
		}

		roleStr := role.(string)
		perms, ok := m.permissions[roleStr]
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Role not defined"})
			return
		}

		hasPermission := false
		for _, p := range perms {
			if p == "*" || p == requiredPermission {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			return
		}

		c.Next()
	}
}
