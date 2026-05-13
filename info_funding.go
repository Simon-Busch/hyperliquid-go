package hyperliquid

import (
	"encoding/json"
	"fmt"
)

// Funding returns historical funding rates for coin in [start, end].
func (i *Info) Funding(coin string, start int64, end *int64) ([]FundingHistory, error) {
	c := i.nameToCoin[coin]
	resp, err := i.postTimeRangeRequest(
		"fundingHistory",
		"",
		start,
		end,
		map[string]any{"coin": c},
	)
	if err != nil {
		return nil, err
	}

	var result []FundingHistory
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal funding history: %w", err)
	}
	return result, nil
}

// UserFunding returns the funding history for addr in [start, end].
func (i *Info) UserFunding(addr string, start int64, end *int64) ([]UserFundingHistory, error) {
	resp, err := i.postTimeRangeRequest("userFunding", addr, start, end, nil)
	if err != nil {
		return nil, err
	}

	var result []UserFundingHistory
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user funding history: %w", err)
	}
	return result, nil
}
