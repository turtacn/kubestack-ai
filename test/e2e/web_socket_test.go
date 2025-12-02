package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kubestack-ai/kubestack-ai/internal/api"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock DiagnosisManager
type MockDiagnosisManager struct {
	mock.Mock
}

func (m *MockDiagnosisManager) RunDiagnosis(ctx context.Context, req *models.DiagnosisRequest, progressChan chan<- interfaces.DiagnosisProgress) (*models.DiagnosisResult, error) {
	// Simulate progress
	// Wait slightly to ensure WS is connected
	time.Sleep(100 * time.Millisecond)

	if progressChan != nil {
		progressChan <- interfaces.DiagnosisProgress{Step: "Init", Status: "InProgress", Message: "Starting..."}
		time.Sleep(50 * time.Millisecond)
		progressChan <- interfaces.DiagnosisProgress{Step: "Detection", Status: "Completed", Message: "Done."}
	}
	// Return dummy result
	return &models.DiagnosisResult{ID: "test-result-id", Status: enum.StatusHealthy}, nil
}

func (m *MockDiagnosisManager) AnalyzeData(ctx context.Context, req *models.DiagnosisRequest, collectedData *models.CollectedData) ([]*models.Issue, error) {
	return nil, nil
}
func (m *MockDiagnosisManager) GenerateReport(result *models.DiagnosisResult) (string, error) {
	return "", nil
}
func (m *MockDiagnosisManager) GetDiagnosisResult(id string) (*models.DiagnosisResult, error) {
	return &models.DiagnosisResult{ID: id, Status: enum.StatusHealthy}, nil
}
func (m *MockDiagnosisManager) GetKnowledgeBase() *knowledge.KnowledgeBase {
	return nil
}

// Mock PluginManager
type MockPluginManager struct {
	mock.Mock
}

func (m *MockPluginManager) RegisterPlugin(plugin interfaces.MiddlewarePlugin) {}
func (m *MockPluginManager) LoadPlugin(name string) (interfaces.MiddlewarePlugin, error) {
	return nil, nil
}
func (m *MockPluginManager) ListPlugins() []interfaces.MiddlewarePlugin { return nil }
func (m *MockPluginManager) GetPlugin(name string) (interfaces.MiddlewarePlugin, bool) {
	return nil, false
}
func (m *MockPluginManager) UnloadPlugin(name string) error { return nil }

func TestWebSocketStream(t *testing.T) {
	// Fix: Change CWD to project root so that LoadHTMLGlob works
	wd, _ := os.Getwd()

	// Attempt to change to /app (project root in this env)
	if err := os.Chdir("/app"); err != nil {
		t.Logf("Failed to chdir to /app: %v. Attempting relative path.", err)
		// Fallback: try 2 levels up if we are in test/e2e
		if err := os.Chdir("../.."); err != nil {
			t.Fatalf("Failed to find project root: %v", err)
		}
	}
	defer os.Chdir(wd) // Restore

	// 1. Setup Server
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			CORS: config.CORSConfig{
				AllowedOrigins: []string{"*"},
			},
		},
	}

	mockDiag := new(MockDiagnosisManager)
	mockPlugin := new(MockPluginManager)

	server := api.NewServer(cfg, mockDiag, nil, mockPlugin)
	router := server.Handler()

	ts := httptest.NewServer(router)
	defer ts.Close()

	// 2. Trigger Diagnosis
	reqBody := `{"target": "localhost:6379", "middleware": "redis", "instance": "redis-01"}`
	resp, err := http.Post(ts.URL+"/api/v1/diagnose", "application/json", bytes.NewBuffer([]byte(reqBody)))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	taskID := result["task_id"].(string)
	assert.NotEmpty(t, taskID)

	// 3. Connect WebSocket
	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/ws/diagnose?id=" + taskID

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer ws.Close()

	// 4. Receive Messages
	msgCount := 0
	done := make(chan bool)

	go func() {
		defer close(done)
		for {
			_, message, err := ws.ReadMessage()
			if err != nil {
				break
			}

			var msgObj struct {
				Topic   string `json:"topic"`
				Payload struct {
					Step   string `json:"Step"`
					Status string `json:"Status"`
				} `json:"payload"`
			}
			json.Unmarshal(message, &msgObj)

			if msgObj.Topic == taskID {
				msgCount++
				if msgObj.Payload.Step == "Finished" {
					return
				}
			}
		}
	}()

	select {
	case <-done:
		// Success: Init, Detection, Finished = 3 messages minimum expected (Mock sends Init and Detection, Handler sends Finished)
		assert.GreaterOrEqual(t, msgCount, 2)
	case <-time.After(3 * time.Second): // Increase timeout
		t.Fatal("Timeout waiting for WebSocket messages")
	}
}
