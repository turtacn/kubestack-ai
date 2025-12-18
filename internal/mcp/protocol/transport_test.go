package protocol

import (
	"strings"
	"testing"
	"time"
)

func TestStdioTransport_SendReceive(t *testing.T) {
	// Use a simple echo command for testing
	transport, err := NewStdioTransport("cat", []string{}, nil)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}
	defer transport.Close()

	// Test sending and receiving
	testMsg := []byte("test message\n")
	if err := transport.Send(testMsg); err != nil {
		t.Fatalf("Failed to send: %v", err)
	}

	received, err := transport.Receive()
	if err != nil {
		t.Fatalf("Failed to receive: %v", err)
	}

	if string(received) != strings.TrimSuffix(string(testMsg), "\n") {
		t.Errorf("Expected %q, got %q", testMsg, received)
	}
}

func TestStdioTransport_Close(t *testing.T) {
	transport, err := NewStdioTransport("cat", []string{}, nil)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	if err := transport.Close(); err != nil {
		t.Errorf("Failed to close: %v", err)
	}

	// Verify transport is closed
	if err := transport.Send([]byte("test")); err == nil {
		t.Error("Expected error when sending to closed transport")
	}
}

func TestStdioTransport_MultipleMessages(t *testing.T) {
	transport, err := NewStdioTransport("cat", []string{}, nil)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}
	defer transport.Close()

	messages := []string{"message1", "message2", "message3"}

	// Send all messages
	for _, msg := range messages {
		if err := transport.Send([]byte(msg + "\n")); err != nil {
			t.Fatalf("Failed to send message %q: %v", msg, err)
		}
	}

	// Receive all messages
	for _, expected := range messages {
		received, err := transport.Receive()
		if err != nil {
			t.Fatalf("Failed to receive: %v", err)
		}

		if string(received) != expected {
			t.Errorf("Expected %q, got %q", expected, received)
		}
	}
}

func TestStdioTransport_AutoNewline(t *testing.T) {
	transport, err := NewStdioTransport("cat", []string{}, nil)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}
	defer transport.Close()

	// Send message without newline
	msg := []byte("test")
	if err := transport.Send(msg); err != nil {
		t.Fatalf("Failed to send: %v", err)
	}

	// Should still receive it
	received, err := transport.Receive()
	if err != nil {
		t.Fatalf("Failed to receive: %v", err)
	}

	if string(received) != string(msg) {
		t.Errorf("Expected %q, got %q", msg, received)
	}
}

func TestStdioTransport_ConcurrentSend(t *testing.T) {
	transport, err := NewStdioTransport("cat", []string{}, nil)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}
	defer transport.Close()

	done := make(chan bool)
	count := 10

	// Send messages concurrently
	for i := 0; i < count; i++ {
		go func(n int) {
			msg := []byte("test\n")
			if err := transport.Send(msg); err != nil {
				t.Errorf("Failed to send: %v", err)
			}
		}(i)
	}

	// Receive messages
	go func() {
		for i := 0; i < count; i++ {
			if _, err := transport.Receive(); err != nil {
				t.Errorf("Failed to receive: %v", err)
				break
			}
		}
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for messages")
	}
}
