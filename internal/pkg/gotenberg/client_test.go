package gotenberg

import (
	"testing"
)

func TestClient_SetHandler(t *testing.T) {
	client := NewClient("http://test")
	handler := &mockStatsHandler{}
	client.SetHandler(handler)

	if client.handler == nil {
		t.Error("Expected handler to be set")
	}
}

func TestClient_GetHandler(t *testing.T) {
	client := NewClient("http://test")
	handler := &mockStatsHandler{}
	client.SetHandler(handler)

	h, ok := client.GetHandler()
	if !ok {
		t.Error("Expected handler to be present")
	}
	if h == nil {
		t.Error("Expected non-nil handler")
	}
}
