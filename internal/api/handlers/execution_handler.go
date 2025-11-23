package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ExecutionHandler struct {
    // manager interfaces.ExecutionManager
}

func NewExecutionHandler() *ExecutionHandler {
	return &ExecutionHandler{}
}

func (h *ExecutionHandler) ExecutePlan(c *gin.Context) {
	id := c.Param("id")
	// TODO: Call ExecutionManager.Execute(id)
	c.JSON(http.StatusOK, gin.H{"status": "executing", "plan_id": id})
}

func (h *ExecutionHandler) GetHistory(c *gin.Context) {
	// TODO: Call ExecutionManager.GetHistory()
	c.JSON(http.StatusOK, gin.H{"history": []string{}})
}
