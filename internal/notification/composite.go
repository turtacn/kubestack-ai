package notification

import (
	"strings"
)

// CompositeNotifier sends notifications using multiple notifiers.
type CompositeNotifier struct {
	notifiers []Notifier
}

// NewCompositeNotifier creates a new CompositeNotifier.
func NewCompositeNotifier(notifiers []Notifier) *CompositeNotifier {
	return &CompositeNotifier{notifiers: notifiers}
}

// Notify sends the notification using all configured notifiers.
// It collects errors from all notifiers but tries to execute all of them.
func (c *CompositeNotifier) Notify(payload *NotificationPayload) error {
	var errs []string
	for _, n := range c.notifiers {
		if err := n.Notify(payload); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return &CompositeError{Errors: errs}
	}
	return nil
}

// CompositeError aggregates multiple errors.
type CompositeError struct {
	Errors []string
}

func (e *CompositeError) Error() string {
	return "notification errors: " + strings.Join(e.Errors, "; ")
}
