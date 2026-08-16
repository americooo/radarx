package notify

import (
	"context"
	"testing"
)

func TestDetectChatIDEmptyToken(t *testing.T) {
	if _, err := DetectChatID(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty token")
	}
}
