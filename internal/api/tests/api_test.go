package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kubestack-ai/kubestack-ai/internal/api"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
    "github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
    "github.com/kubestack-ai/kubestack-ai/internal/core/models"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "context"
)

// MockDiagnosisManager
type MockDiagnosisManager struct {
    mock.Mock
}

func (m *MockDiagnosisManager) RunDiagnosis(ctx context.Context, req *models.DiagnosisRequest, progressChan chan<- interfaces.DiagnosisProgress) (*models.DiagnosisResult, error) {
    args := m.Called(ctx, req, progressChan)
    // Simulate progress
    if progressChan != nil {
        progressChan <- interfaces.DiagnosisProgress{Status: "Completed"}
    }
    return args.Get(0).(*models.DiagnosisResult), args.Error(1)
}

func (m *MockDiagnosisManager) GenerateReport(result *models.DiagnosisResult) (string, error) {
    return "report", nil
}

func (m *MockDiagnosisManager) AnalyzeData(ctx context.Context, req *models.DiagnosisRequest, collectedData *models.CollectedData) ([]*models.Issue, error) {
    return nil, nil
}

func TestDiagnosisAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)

    // Config
    cfg := &config.Config{
        Auth: config.AuthConfig{JWTSecret: "secret", TokenTTL: time.Hour},
        RBAC: config.RBACConfig{Roles: map[string]config.RoleConfig{"admin": {Permissions: []string{"*"}}}},
        WebSocket: config.WebSocketConfig{},
        Server: config.ServerConfig{Port: 8080},
    }
    logger.InitGlobalLogger(&logger.Config{Level: "error"})

    // Mocks
    mockEngine := new(MockDiagnosisManager)
    mockEngine.On("RunDiagnosis", mock.Anything, mock.Anything, mock.Anything).Return(&models.DiagnosisResult{ID: "test-id"}, nil)

	server := api.NewServer(cfg, mockEngine)

    // 1. Login to get token
    loginBody := []byte(`{"username": "admin", "password": "admin"}`)
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody))
    server.Handler().ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    var loginResp map[string]string
    json.Unmarshal(w.Body.Bytes(), &loginResp)
    token := loginResp["token"]
    assert.NotEmpty(t, token)

    // 2. Trigger Diagnosis
    diagBody := []byte(`{"target": "redis", "middleware": "redis"}`)
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("POST", "/api/v1/diagnosis", bytes.NewBuffer(diagBody))
    req.Header.Set("Authorization", "Bearer " + token)
    server.Handler().ServeHTTP(w, req)

    assert.Equal(t, http.StatusAccepted, w.Code)
}
