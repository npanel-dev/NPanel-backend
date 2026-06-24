package server

import (
	"net/http"
	"strings"
	"testing"

	adminsubscribev1 "github.com/npanel-dev/NPanel-backend/api/admin/subscribe/v1"
)

func TestCustomRequestDecoderNormalizesEmptyNodeGroupID(t *testing.T) {
	body := `{"name":"Test","unit_price":10000,"unit_time":"Month","discount":[],"inventory":-1,"speed_limit":0,"device_limit":0,"traffic":107374182400,"quota":0,"language":"","node_tags":[],"nodes":["1"],"node_group_id":"","node_group_ids":[],"deduction_ratio":0,"reset_cycle":0,"renewal_reset":false,"show_original_price":false,"traffic_limit":[],"show":false,"sell":false}`
	req := httptestRequest(http.MethodPost, "/v1/admin/subscribe", body)
	var decoded adminsubscribev1.CreateSubscribeRequest

	if err := CustomRequestDecoder(req, &decoded); err != nil {
		t.Fatalf("CustomRequestDecoder returned error: %v", err)
	}

	if decoded.NodeGroupId != 0 {
		t.Fatalf("NodeGroupId = %d, want 0", decoded.NodeGroupId)
	}
	if len(decoded.Nodes) != 1 || decoded.Nodes[0] != 1 {
		t.Fatalf("Nodes = %v, want [1]", decoded.Nodes)
	}
}

func TestCustomRequestDecoderKeepsNumericNodeGroupIDString(t *testing.T) {
	req := httptestRequest(http.MethodPut, "/v1/admin/subscribe", `{"id":1,"node_group_id":"7"}`)
	var decoded adminsubscribev1.UpdateSubscribeRequest

	if err := CustomRequestDecoder(req, &decoded); err != nil {
		t.Fatalf("CustomRequestDecoder returned error: %v", err)
	}

	if decoded.NodeGroupId != 7 {
		t.Fatalf("NodeGroupId = %d, want 7", decoded.NodeGroupId)
	}
}

func TestCustomRequestDecoderNormalizesCamelCaseEmptyNodeGroupID(t *testing.T) {
	req := httptestRequest(http.MethodPost, "/v1/admin/subscribe", `{"nodeGroupId":""}`)
	var decoded adminsubscribev1.CreateSubscribeRequest

	if err := CustomRequestDecoder(req, &decoded); err != nil {
		t.Fatalf("CustomRequestDecoder returned error: %v", err)
	}

	if decoded.NodeGroupId != 0 {
		t.Fatalf("NodeGroupId = %d, want 0", decoded.NodeGroupId)
	}
}

func httptestRequest(method, path, body string) *http.Request {
	req, _ := http.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}
