package server

import (
	"encoding/json"
	"testing"
)

func TestCompatLegacyOnlineUserAcceptsSIDAlias(t *testing.T) {
	var req compatLegacyPushOnlineUsersRequest
	if err := json.Unmarshal([]byte(`{"users":[{"sid":3,"ip":"1.54.7.44"}]}`), &req); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if len(req.Users) != 1 {
		t.Fatalf("Users length = %d, want 1", len(req.Users))
	}
	if req.Users[0].SID != 3 {
		t.Fatalf("SID = %d, want 3", req.Users[0].SID)
	}
	if req.Users[0].IP != "1.54.7.44" {
		t.Fatalf("IP = %q, want 1.54.7.44", req.Users[0].IP)
	}
}
