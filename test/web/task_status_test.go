package web_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"github.com/kubestack-ai/kubestack-ai/internal/storage"
	"github.com/kubestack-ai/kubestack-ai/internal/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock objects need to be defined or generated.
// Since we can't easily generate mocks here, I'll define minimal mocks.

type MockDiagnosisManager struct {
	mock.Mock
}

func (m *MockDiagnosisManager) RunDiagnosis(ctx context.Context, req *models.DiagnosisRequest, progressChan chan<- interfaces.DiagnosisProgress) (*models.DiagnosisResult, error) {
	args := m.Called(ctx, req, progressChan)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiagnosisResult), args.Error(1)
}
func (m *MockDiagnosisManager) AnalyzeData(ctx context.Context, req *models.DiagnosisRequest, collectedData *models.CollectedData) ([]*models.Issue, error) {
	return nil, nil
}
func (m *MockDiagnosisManager) GenerateReport(result *models.DiagnosisResult) (string, error) {
	return "", nil
}
func (m *MockDiagnosisManager) GetDiagnosisResult(id string) (*models.DiagnosisResult, error) {
	return nil, nil
}

func TestTaskStatusAPI(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	taskStore := storage.NewInMemoryTaskStore()
	taskID := "task-100"
	taskStore.CreateTask(taskID)
	taskStore.UpdateStatus(taskID, storage.TaskStateCompleted)
	taskStore.SaveResult(taskID, &models.DiagnosisResult{ID: taskID, Summary: "Done"})

	handler := web.NewConsoleHandler(nil, nil, taskStore)
	handler.RegisterRoutes(router)

	// Action
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/console/task/status/"+taskID, nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, taskID, response["task_id"])
	assert.Equal(t, string(storage.TaskStateCompleted), response["state"])

	result, ok := response["result"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "Done", result["summary"])
}
