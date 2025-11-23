package notification_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubestack-ai/kubestack-ai/internal/notification"
	"github.com/stretchr/testify/assert"
)

func TestWebhookNotification(t *testing.T) {
	// Setup: Mock Webhook Server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		var payload notification.NotificationPayload
		json.Unmarshal(body, &payload)

		assert.Equal(t, "task-3", payload.TaskID)
		assert.Equal(t, "COMPLETED", payload.Status)

		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	// Action: Trigger notification
	notifier := notification.NewWebhookNotifier(mockServer.URL)
	err := notifier.Notify(&notification.NotificationPayload{TaskID: "task-3", Status: "COMPLETED"})

	// Assert
	assert.NoError(t, err)
}
