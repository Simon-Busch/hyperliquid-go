package hyperliquid

import (
	"encoding/json"
	"testing"
)

func TestResultFromResponse_Nil(t *testing.T) {
	r := resultFromResponse(nil)
	if r != (Result{}) {
		t.Errorf("nil response should yield zero Result, got %+v", r)
	}
}

func TestResultFromResponse_Error(t *testing.T) {
	resp := &APIResponse[OrderResponse]{Ok: false, Err: "boom"}
	r := resultFromResponse(resp)
	if r.Error != "boom" {
		t.Errorf("Error = %q", r.Error)
	}
}

func TestResultFromResponse_RestingOrder(t *testing.T) {
	// JSON field for cloid on the wire is "cid".
	raw := json.RawMessage(`{"resting":{"oid":42,"cid":"0xabc","status":"open"}}`)
	resp := &APIResponse[OrderResponse]{Ok: true, Data: OrderResponse{Statuses: MixedArray{
		MixedValue(raw),
	}}}
	r := resultFromResponse(resp)
	if r.OID != 42 || r.Cloid != "0xabc" || r.Status != "open" {
		t.Errorf("resting parse mismatch: %+v", r)
	}
}

func TestResultFromResponse_FilledOrder(t *testing.T) {
	raw := json.RawMessage(`{"filled":{"oid":7,"avgPx":"100.5","totalSz":"0.5"}}`)
	resp := &APIResponse[OrderResponse]{Ok: true, Data: OrderResponse{Statuses: MixedArray{
		MixedValue(raw),
	}}}
	r := resultFromResponse(resp)
	if r.OID != 7 || r.AvgPx != "100.5" || r.TotalSz != "0.5" || r.Status != "filled" {
		t.Errorf("filled parse mismatch: %+v", r)
	}
}

func TestResultFromResponse_StringStatus(t *testing.T) {
	// String statuses are forwarded as the raw JSON value (including
	// surrounding quotes) — this locks the current behaviour.
	raw := json.RawMessage(`"waitingForOpen"`)
	resp := &APIResponse[OrderResponse]{Ok: true, Data: OrderResponse{Statuses: MixedArray{
		MixedValue(raw),
	}}}
	r := resultFromResponse(resp)
	if r.Status != `"waitingForOpen"` {
		t.Errorf("string status = %q", r.Status)
	}
}

func TestBatchResultFromResponse_Multi(t *testing.T) {
	resp := &APIResponse[OrderResponse]{Ok: true, Data: OrderResponse{Statuses: MixedArray{
		MixedValue(`{"resting":{"oid":1,"cid":"","status":"open"}}`),
		MixedValue(`{"filled":{"oid":2,"avgPx":"100","totalSz":"1"}}`),
	}}}
	br := batchResultFromResponse(resp)
	if len(br.Results) != 2 {
		t.Fatalf("len(Results) = %d, want 2", len(br.Results))
	}
	if br.Results[0].OID != 1 || br.Results[0].Status != "open" {
		t.Errorf("leg 0: %+v", br.Results[0])
	}
	if br.Results[1].OID != 2 || br.Results[1].Status != "filled" {
		t.Errorf("leg 1: %+v", br.Results[1])
	}
}

func TestBatchResultFromResponse_Error(t *testing.T) {
	resp := &APIResponse[OrderResponse]{Ok: false, Err: "rejected"}
	br := batchResultFromResponse(resp)
	if br.Error != "rejected" {
		t.Errorf("Error = %q", br.Error)
	}
}

func TestFirstCancelResult_Nil(t *testing.T) {
	if r := firstCancelResult(nil); r != (CancelResult{}) {
		t.Errorf("expected zero, got %+v", r)
	}
}

func TestFirstCancelResult_Error(t *testing.T) {
	r := firstCancelResult(&APIResponse[CancelOrderResponse]{Ok: false, Err: "bad oid"})
	if r.Error != "bad oid" {
		t.Errorf("Error = %q", r.Error)
	}
}

func TestFirstCancelResult_Status(t *testing.T) {
	resp := &APIResponse[CancelOrderResponse]{Ok: true, Data: CancelOrderResponse{Statuses: MixedArray{
		MixedValue(`"success"`),
	}}}
	r := firstCancelResult(resp)
	// firstCancelResult stringifies the raw MixedValue verbatim, so the
	// surrounding JSON quotes are preserved.
	if r.Status != `"success"` {
		t.Errorf("Status = %q", r.Status)
	}
}

func TestCancelBatchFromResponse_Multi(t *testing.T) {
	resp := &APIResponse[CancelOrderResponse]{Ok: true, Data: CancelOrderResponse{Statuses: MixedArray{
		MixedValue(`"success"`),
		MixedValue(`"success"`),
	}}}
	br := cancelBatchFromResponse(resp)
	if len(br.Results) != 2 {
		t.Errorf("len(Results) = %d", len(br.Results))
	}
}
