package hyperliquid

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// OutcomeGroup is the HIP-4 user-outcome subgroup on Trader. The four
// methods mint or burn outcome shares against USDH without touching the
// order book — they're the prediction-market equivalent of conversion
// between collateral and synthetic positions.
//
//   - Split:        X USDH -> X Yes + X No of one outcome
//   - Merge:        X Yes + X No of one outcome -> X USDH (amount nil = max)
//   - MergeQuestion: X Yes of every named outcome -> X USDH (amount nil = max)
//   - Negate:       X No of one bucket -> X Yes of every OTHER bucket in
//                   the same question
//
// Every variant is L1-signed and posted to /exchange under
// type:"userOutcome"; HL parses by the inner body key.
type OutcomeGroup struct {
	t *Trader
}

// Split mints `amount` Yes shares and `amount` No shares of `outcome`
// by burning `amount` USDH from the caller's wallet.
func (g *OutcomeGroup) Split(outcome uint64, amount float64) (*TransferResponse, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("split: amount must be > 0, got %v", amount)
	}
	action := SplitOutcomeAction{
		Type: "userOutcome",
		SplitOutcome: SplitOutcomeWire{
			Outcome: outcome,
			Amount:  formatUsdAmount(amount),
		},
	}
	return g.submit(action)
}

// Merge burns `amount` Yes + `amount` No of `outcome` back into
// `amount` USDH. Pass nil to burn the maximum holdable (min(Yes, No)).
func (g *OutcomeGroup) Merge(outcome uint64, amount *float64) (*TransferResponse, error) {
	wire := MergeOutcomeWire{Outcome: outcome}
	if amount != nil {
		if *amount <= 0 {
			return nil, fmt.Errorf("merge: amount must be > 0 (pass nil for max), got %v", *amount)
		}
		s := formatUsdAmount(*amount)
		wire.Amount = &s
	}
	action := MergeOutcomeAction{Type: "userOutcome", MergeOutcome: wire}
	return g.submit(action)
}

// MergeQuestion burns `amount` Yes from each named outcome of `question`
// into `amount` USDH. Since exactly one bucket of a question resolves
// true, holding one Yes per bucket guarantees one USDH payout — Merge
// realises that early. Pass nil to burn the maximum (the min Yes
// balance across buckets).
func (g *OutcomeGroup) MergeQuestion(question uint64, amount *float64) (*TransferResponse, error) {
	wire := MergeQuestionWire{Question: question}
	if amount != nil {
		if *amount <= 0 {
			return nil, fmt.Errorf("mergeQuestion: amount must be > 0 (pass nil for max), got %v", *amount)
		}
		s := formatUsdAmount(*amount)
		wire.Amount = &s
	}
	action := MergeQuestionAction{Type: "userOutcome", MergeQuestion: wire}
	return g.submit(action)
}

// Negate converts `amount` No shares of `outcome` (a bucket of
// `question`) into `amount` Yes shares of every OTHER bucket in the
// same question. The total share count grows by (numBuckets-2) * amount;
// the USDH-equivalent value is preserved by the protocol.
func (g *OutcomeGroup) Negate(question, outcome uint64, amount float64) (*TransferResponse, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("negate: amount must be > 0, got %v", amount)
	}
	action := NegateOutcomeAction{
		Type: "userOutcome",
		NegateOutcome: NegateOutcomeWire{
			Question: question,
			Outcome:  outcome,
			Amount:   formatUsdAmount(amount),
		},
	}
	return g.submit(action)
}

// submit signs the userOutcome action as L1 and posts it. Surfaces a
// venue-side error ({"status":"err","response":"..."}) as a Go error
// so callers don't have to inspect the response shape manually.
func (g *OutcomeGroup) submit(action any) (*TransferResponse, error) {
	var result TransferResponse
	if err := g.t.executeAction(action, &result); err != nil {
		return nil, err
	}
	if result.Status != "" && result.Status != "ok" {
		// On error responses Hyperliquid puts the message in the
		// `response` field as a bare JSON string. The TransferResponse
		// struct already preserves the raw RawMessage; surface it.
		var reason string
		_ = json.Unmarshal(result.Response, &reason)
		if reason == "" {
			reason = result.Status
		}
		return &result, fmt.Errorf("userOutcome rejected: %s", reason)
	}
	return &result, nil
}

// outcomeAssetIDForCanonical parses "#<enc>" canonical names into the
// numeric outcome id used by the userOutcome wire actions. enc =
// 10*outcome + sideIdx, so the outcome id is enc / 10. Returns an
// error when the name isn't in canonical form.
func outcomeIDFromCanonical(canonical string) (uint64, error) {
	if len(canonical) < 2 || canonical[0] != '#' {
		return 0, fmt.Errorf("outcome id: expected canonical \"#<enc>\", got %q", canonical)
	}
	enc, err := strconv.ParseUint(canonical[1:], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("outcome id: parse %q: %w", canonical, err)
	}
	return enc / 10, nil
}
