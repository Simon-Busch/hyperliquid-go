package info

import (
	"fmt"
	"strings"
)

// ParseOutcomeDescription splits a HIP-4 description string into its
// constituent key/value pairs. Hyperliquid encodes outcome and question
// metadata as "class:priceBinary|underlying:BTC|expiry:..." — a flat
// pipe-delimited map. Unknown shapes return an empty map; the function
// never panics on malformed input.
func ParseOutcomeDescription(s string) map[string]string {
	out := map[string]string{}
	if s == "" {
		return out
	}
	for _, kv := range strings.Split(s, "|") {
		idx := strings.Index(kv, ":")
		if idx <= 0 || idx == len(kv)-1 {
			continue
		}
		out[kv[:idx]] = kv[idx+1:]
	}
	return out
}

// BucketLabels derives a human-readable label for every entry in
// q.NamedOutcomes. For class:priceBucket questions with thresholds
// [T1, T2, ..., Tn] and n+1 named outcomes the labels are
// ["Below T1", "T1 to T2", ..., "Above Tn"].
//
// Non price-bucket questions, or questions whose named-outcome count
// doesn't match the threshold count + 1, return nil — callers should
// fall back to OutcomeInfo.Name + ":" + sideSpec.Name in that case.
func (q Question) BucketLabels() []string {
	desc := ParseOutcomeDescription(q.Description)
	if desc["class"] != "priceBucket" {
		return nil
	}
	raw := desc["priceThresholds"]
	if raw == "" {
		return nil
	}
	thresholds := strings.Split(raw, ",")
	if len(thresholds) == 0 {
		return nil
	}
	if len(q.NamedOutcomes) != len(thresholds)+1 {
		return nil
	}
	labels := make([]string, len(q.NamedOutcomes))
	labels[0] = "Below " + thresholds[0]
	for i := 1; i < len(thresholds); i++ {
		labels[i] = thresholds[i-1] + " to " + thresholds[i]
	}
	labels[len(thresholds)] = "Above " + thresholds[len(thresholds)-1]
	return labels
}

// FindQuestion locates the Question that owns outcome by walking
// NamedOutcomes and FallbackOutcome. Returns nil when no question
// references the outcome (binary markets like a class:priceBinary
// outcome stand alone).
func (m *OutcomeMeta) FindQuestion(outcome int) *Question {
	if m == nil {
		return nil
	}
	for i := range m.Questions {
		q := &m.Questions[i]
		if q.FallbackOutcome == outcome {
			return q
		}
		for _, o := range q.NamedOutcomes {
			if o == outcome {
				return q
			}
		}
	}
	return nil
}

// BucketLabel returns the human-readable bucket name for outcome
// within its parent question, or the empty string when outcome is
// not a named bucket of a price-bucket question.
func (m *OutcomeMeta) BucketLabel(outcome int) string {
	q := m.FindQuestion(outcome)
	if q == nil {
		return ""
	}
	labels := q.BucketLabels()
	if labels == nil {
		return ""
	}
	for i, o := range q.NamedOutcomes {
		if o == outcome {
			return labels[i]
		}
	}
	if outcome == q.FallbackOutcome {
		return q.Name + " fallback"
	}
	return ""
}

// QuestionByName returns the first Question whose Name matches.
func (m *OutcomeMeta) QuestionByName(name string) *Question {
	if m == nil {
		return nil
	}
	for i := range m.Questions {
		if m.Questions[i].Name == name {
			return &m.Questions[i]
		}
	}
	return nil
}

// Buckets returns one entry per named outcome of q, ready for trading.
// The Canonical name (`#<10*outcome+sideIdx>`) is what PlaceMarket and
// friends expect.
type Bucket struct {
	Outcome      int    // outcome id within the bucket
	Label        string // human label, e.g. "75348 to 78423"
	YesCanonical string // "#<10*outcome+0>"
	NoCanonical  string // "#<10*outcome+1>"
}

// Buckets returns one Bucket per entry in q.NamedOutcomes. Returns
// nil when BucketLabels cannot derive labels — the caller probably
// wants to treat the question as opaque in that case.
func (q Question) Buckets() []Bucket {
	labels := q.BucketLabels()
	if labels == nil {
		return nil
	}
	out := make([]Bucket, len(q.NamedOutcomes))
	for i, o := range q.NamedOutcomes {
		out[i] = Bucket{
			Outcome:      o,
			Label:        labels[i],
			YesCanonical: fmt.Sprintf("#%d", 10*o+0),
			NoCanonical:  fmt.Sprintf("#%d", 10*o+1),
		}
	}
	return out
}
