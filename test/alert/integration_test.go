package alert_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/kubestack-ai/kubestack-ai/internal/alert"
	"github.com/kubestack-ai/kubestack-ai/internal/alert/notifier"
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
	"github.com/kubestack-ai/kubestack-ai/internal/core/diagnosis"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// Mock objects need to be defined or we use what's available.
// Since we are inside test/alert package, we need to import internal packages.
// Mocking Manager is hard because it's a struct, not interface.
// But NewManager takes interfaces. We can mock interfaces passed to NewManager.
// And Dispatcher takes *diagnosis.Manager.

// We will test Correlator and Dispatcher logic mostly.

func TestCorrelator_MergeAlerts(t *testing.T) {
	var flushed *alert.CorrelatedAlert
	var wg sync.WaitGroup
	wg.Add(1)

	onFlush := func(a *alert.CorrelatedAlert) {
		flushed = a
		wg.Done()
	}

	// Short window for testing
	c := alert.NewCorrelator(100*time.Millisecond, onFlush)

	// Add 3 related alerts
	now := time.Now()
	e1 := &models.AlertEvent{Instance: "redis-1", Name: "HighCPU", Severity: enum.SeverityWarning, StartsAt: now}
	e2 := &models.AlertEvent{Instance: "redis-1", Name: "HighMem", Severity: enum.SeverityWarning, StartsAt: now}
	e3 := &models.AlertEvent{Instance: "redis-1", Name: "SlowLog", Severity: enum.SeverityWarning, StartsAt: now}

	c.AddAlert(e1)
	c.AddAlert(e2)
	c.AddAlert(e3)

	// Wait for flush
	wg.Wait()

	assert.NotNil(t, flushed)
	assert.Equal(t, "redis-1", flushed.Instance)
	assert.Equal(t, 3, len(flushed.Alerts))
}

type MockNotifier struct {
	Sent []*notifier.NotificationMessage
	mu   sync.Mutex
}

func (m *MockNotifier) Send(ctx context.Context, msg *notifier.NotificationMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Sent = append(m.Sent, msg)
	return nil
}

func (m *MockNotifier) Type() string { return "mock" }

func TestFeedbackProcessor(t *testing.T) {
	mockNotif := &MockNotifier{}
	fp := alert.NewFeedbackProcessor([]notifier.Notifier{mockNotif}, &alert.FeedbackConfig{})

	correlated := &alert.CorrelatedAlert{
		Instance:   "mysql-prod",
		Middleware: enum.MySQL,
		Summary:    "Connection spike",
	}

	result := &models.DiagnosisResult{
		Status:  enum.StatusCritical,
		Summary: "Too many sleep connections",
		Issues: []*models.Issue{
			{Title: "Sleep Connections", Severity: enum.SeverityCritical, Description: "1000 sleep connections"},
		},
	}

	err := fp.ProcessDiagnosisResult(context.Background(), correlated, result)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(mockNotif.Sent))
	assert.Contains(t, mockNotif.Sent[0].Title, "Critical")
	assert.Contains(t, mockNotif.Sent[0].Content, "mysql-prod")
}

// Integration test for flow Dispatcher -> Diagnosis -> Feedback
// We need to mock PluginManager to pass to DiagnosisManager.

type MockPluginManager struct {
	mock.Mock
}

func (m *MockPluginManager) LoadPlugins() error {
	return nil
}
func (m *MockPluginManager) GetPlugin(name string) (interfaces.DiagnosticPlugin, error) {
	return nil, nil // Not used in this flow if we mock CollectData
}
func (m *MockPluginManager) ListPlugins() []interfaces.DiagnosticPlugin {
	return nil
}
func (m *MockPluginManager) CollectData(ctx context.Context, req *models.DiagnosisRequest) (*models.CollectedData, error) {
	return &models.CollectedData{}, nil // Return empty data
}
func (m *MockPluginManager) Shutdown() {
}
func (m *MockPluginManager) LoadPlugin(pluginName string) (interfaces.DiagnosticPlugin, error) {
	return nil, nil
}
func (m *MockPluginManager) UnloadPlugin(pluginName string) error {
	return nil
}

func TestDispatcher_Trigger(t *testing.T) {
	// Setup mocks
	pm := new(MockPluginManager)
	// We need a real manager or mock?
	// Since Dispatcher takes *Manager struct, we must use real Manager.
	// We pass mock PM to it.

	dm := diagnosis.NewManager(pm, nil, nil, "", nil)

	mockNotif := &MockNotifier{}
	feedback := alert.NewFeedbackProcessor([]notifier.Notifier{mockNotif}, &alert.FeedbackConfig{})

	c := alert.NewCorrelator(100*time.Millisecond, nil)

	d := alert.NewDispatcher(dm, c, feedback, &alert.DispatcherConfig{})

	// Create a correlated alert manually and trigger
	correlated := &alert.CorrelatedAlert{
		Instance: "test-instance",
		Middleware: enum.Redis,
	}

	// Trigger
	err := d.TriggerDiagnosis(context.Background(), correlated)
	assert.NoError(t, err)

	// Wait a bit for async execution
	time.Sleep(200 * time.Millisecond)

	// Check feedback
	assert.Equal(t, 1, len(mockNotif.Sent))
	assert.Contains(t, mockNotif.Sent[0].Content, "test-instance")
}

func TestDingTalkNotifier(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		// Check query params if secret is set
		q := r.URL.Query()
		if q.Get("sign") == "" {
			t.Error("missing signature")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n := notifier.NewDingTalkNotifier(server.URL, "secret")
	err := n.Send(context.Background(), &notifier.NotificationMessage{
		Title: "Test",
		Content: "Hello",
	})
	assert.NoError(t, err)
}
