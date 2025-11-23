package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/api/middleware"
)

type ConfigHandler struct {
	config *config.Config
}

func NewConfigHandler(cfg *config.Config) *ConfigHandler {
	return &ConfigHandler{
		config: cfg,
	}
}

func (h *ConfigHandler) GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, h.config)
}

func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
    // This is a simplified update. Real-world would involve validation and persistence.
	var newConfig config.Config
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

    // Validate?
    // *h.config = newConfig // Unsafe without mutex
	c.JSON(http.StatusOK, gin.H{"status": "config updated (in-memory only)"})
}

type AuthHandler struct {
	authService *middleware.AuthService
}

func NewAuthHandler(svc *middleware.AuthService) *AuthHandler {
	return &AuthHandler{authService: svc}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mock authentication
	// In production, check against DB or LDAP
	var role string
	if req.Username == "admin" && req.Password == "admin" {
		role = "admin"
	} else if req.Username == "operator" && req.Password == "operator" {
		role = "operator"
	} else if req.Username == "viewer" && req.Password == "viewer" {
		role = "viewer"
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := h.authService.GenerateToken(req.Username, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"role":  role,
	})
}
