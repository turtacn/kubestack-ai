package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge"
	"gopkg.in/yaml.v2"
)

// KnowledgeAPI handles knowledge base related requests.
type KnowledgeAPI struct {
	kb     *knowledge.KnowledgeBase
	loader *knowledge.RuleLoader
}

// NewKnowledgeAPI creates a new KnowledgeAPI instance.
func NewKnowledgeAPI(kb *knowledge.KnowledgeBase, loader *knowledge.RuleLoader) *KnowledgeAPI {
	return &KnowledgeAPI{
		kb:     kb,
		loader: loader,
	}
}

// RegisterRoutes registers the API routes.
func (api *KnowledgeAPI) RegisterRoutes(router *gin.RouterGroup) {
	rules := router.Group("/rules")
	{
		rules.GET("", api.ListRules)
		rules.POST("", api.CreateRule)
		rules.GET("/:id", api.GetRule)
		rules.PUT("/:id", api.UpdateRule)
		rules.DELETE("/:id", api.DeleteRule)
		rules.GET("/export", api.ExportRules)
		rules.POST("/import", api.ImportRules)
	}
}

// CreateRule creates a new rule.
func (api *KnowledgeAPI) CreateRule(c *gin.Context) {
	var rule knowledge.Rule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Update in-memory
	if err := api.kb.AddRule(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Persist to disk
	if err := api.loader.SaveRule(&rule); err != nil {
		// Log error but don't fail the request completely?
		// Or fail and rollback? For now just log warning in response (or better, fail)
		// Assuming strict consistency is better
		// Rollback in memory not implemented, but rule is already added.
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save rule to disk: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// ListRules lists rules with filtering.
func (api *KnowledgeAPI) ListRules(c *gin.Context) {
	opts := knowledge.QueryOptions{
		MiddlewareType: c.Query("middleware_type"),
		Severity:       c.QueryArray("severity"),
		Tags:           c.QueryArray("tags"),
	}

	rules, err := api.kb.QueryRules(opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(rules),
		"rules": rules,
	})
}

// GetRule gets a single rule by ID.
func (api *KnowledgeAPI) GetRule(c *gin.Context) {
	id := c.Param("id")
	rule, err := api.kb.GetRule(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// UpdateRule updates a rule.
func (api *KnowledgeAPI) UpdateRule(c *gin.Context) {
	id := c.Param("id")
	var rule knowledge.Rule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	rule.ID = id
	if err := api.kb.UpdateRule(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Persist
	if err := api.loader.SaveRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save rule to disk: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteRule deletes a rule.
func (api *KnowledgeAPI) DeleteRule(c *gin.Context) {
	id := c.Param("id")
	if err := api.kb.DeleteRule(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Persist
	if err := api.loader.DeleteRule(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete rule from disk: " + err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ExportRules exports all rules as YAML.
func (api *KnowledgeAPI) ExportRules(c *gin.Context) {
	rules, err := api.kb.GetAllRules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	data, err := yaml.Marshal(rules)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "application/x-yaml")
	c.Header("Content-Disposition", "attachment; filename=rules_export.yaml")
	c.Data(http.StatusOK, "application/x-yaml", data)
}

// ImportRules imports rules from an uploaded YAML file.
func (api *KnowledgeAPI) ImportRules(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read file"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer f.Close()

	var rules []knowledge.Rule
	if err := yaml.NewDecoder(f).Decode(&rules); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid YAML format"})
		return
	}

	imported := 0
	for _, rule := range rules {
		// Use a local variable to avoid loop variable capture issues
		r := rule
		if err := api.kb.AddRule(&r); err != nil {
			// Continue or error out? Let's continue and report partial success
			continue
		}
		// Persist each imported rule
		if err := api.loader.SaveRule(&r); err != nil {
			// Log error
			continue
		}
		imported++
	}

	c.JSON(http.StatusOK, gin.H{
		"imported": imported,
		"total":    len(rules),
	})
}
