package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	adminsubscribev1 "github.com/npanel-dev/NPanel-backend/api/admin/subscribe/v1"
)

// CustomRequestDecoder keeps compatibility with legacy admin forms that submit
// an empty node_group_id string when no default node group is selected.
func CustomRequestDecoder(r *http.Request, v interface{}) error {
	if !shouldNormalizeEmptyNodeGroupID(v) {
		return khttp.DefaultRequestDecoder(r, v)
	}

	data, err := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(data))
	if err != nil {
		return errors.BadRequest("CODEC", err.Error())
	}
	if len(data) == 0 {
		return khttp.DefaultRequestDecoder(r, v)
	}

	normalized, ok := normalizeEmptyNodeGroupID(data)
	if ok {
		r.Body = io.NopCloser(bytes.NewBuffer(normalized))
	}

	return khttp.DefaultRequestDecoder(r, v)
}

func shouldNormalizeEmptyNodeGroupID(v interface{}) bool {
	switch v.(type) {
	case *adminsubscribev1.CreateSubscribeRequest, *adminsubscribev1.UpdateSubscribeRequest:
		return true
	default:
		return false
	}
}

func normalizeEmptyNodeGroupID(data []byte) ([]byte, bool) {
	var body map[string]json.RawMessage
	if err := json.Unmarshal(data, &body); err != nil {
		return data, false
	}

	changed := normalizeEmptyStringInt64Field(body, "node_group_id")
	changed = normalizeEmptyStringInt64Field(body, "nodeGroupId") || changed
	if !changed {
		return data, false
	}

	normalized, err := json.Marshal(body)
	if err != nil {
		return data, false
	}
	return normalized, true
}

func normalizeEmptyStringInt64Field(body map[string]json.RawMessage, field string) bool {
	raw, ok := body[field]
	if !ok {
		return false
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	if strings.TrimSpace(value) != "" {
		return false
	}

	body[field] = json.RawMessage("0")
	return true
}
