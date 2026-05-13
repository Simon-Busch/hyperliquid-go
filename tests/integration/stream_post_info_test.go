//go:build integration

package integration

import (
	"encoding/json"
	"testing"
	"time"
)

// TestStream_PostInfo_MatchesREST issues a Meta request over the WS
// PostInfo channel and compares the result to a REST Info.Meta call.
// Both must return the same number of perp assets.
func TestStream_PostInfo_MatchesREST(t *testing.T) {
	c := newStreamingClient(t)

	restMeta, err := c.Info.Meta()
	if err != nil {
		t.Fatalf("REST Meta: %v", err)
	}

	raw, err := c.Stream.PostInfo(map[string]any{"type": "meta"}, 10*time.Second)
	if err != nil {
		t.Fatalf("Stream.PostInfo: %v", err)
	}
	var wsMeta struct {
		Universe []map[string]any `json:"universe"`
	}
	if err := json.Unmarshal(raw, &wsMeta); err != nil {
		t.Fatalf("unmarshal WS meta: %v", err)
	}
	if len(wsMeta.Universe) != len(restMeta.Universe) {
		t.Errorf("universe length mismatch: ws=%d rest=%d", len(wsMeta.Universe), len(restMeta.Universe))
	}
}
