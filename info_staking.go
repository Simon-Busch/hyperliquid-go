package hyperliquid

import (
	"encoding/json"
	"fmt"
)

// InfoStakeGroup exposes the staking-info shortcuts. Accessed via the
// Info.Stake field, populated by NewInfo.
type InfoStakeGroup struct{ i *Info }

// Summary returns the staking summary for addr.
func (g *InfoStakeGroup) Summary(addr string) (*StakingSummary, error) {
	resp, err := g.i.client.post("/info", map[string]any{
		"type": "delegatorSummary",
		"user": addr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch staking summary: %w", err)
	}

	var result StakingSummary
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal staking summary: %w", err)
	}
	return &result, nil
}

// Delegations returns the active delegations for addr.
func (g *InfoStakeGroup) Delegations(addr string) ([]StakingDelegation, error) {
	resp, err := g.i.client.post("/info", map[string]any{
		"type": "delegations",
		"user": addr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch staking delegations: %w", err)
	}

	var result []StakingDelegation
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal staking delegations: %w", err)
	}
	return result, nil
}

// Rewards returns the staking reward history for addr.
func (g *InfoStakeGroup) Rewards(addr string) ([]StakingReward, error) {
	resp, err := g.i.client.post("/info", map[string]any{
		"type": "delegatorRewards",
		"user": addr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch staking rewards: %w", err)
	}

	var result []StakingReward
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal staking rewards: %w", err)
	}
	return result, nil
}
