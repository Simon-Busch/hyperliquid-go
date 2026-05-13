package hyperliquid

import (
	"encoding/json"
	"fmt"
)

// Funding returns historical funding rates for coin in [start, end].
func (i *Info) Funding(coin string, start int64, end *int64) ([]FundingHistory, error) {
	return i.FundingHistory(coin, start, end)
}

// FundingHistory fetches the historical funding rate stream for name.
func (i *Info) FundingHistory(
	name string,
	startTime int64,
	endTime *int64,
) ([]FundingHistory, error) {
	coin := i.nameToCoin[name]
	resp, err := i.postTimeRangeRequest(
		"fundingHistory",
		"",
		startTime,
		endTime,
		map[string]any{"coin": coin},
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
	return i.UserFundingHistory(addr, start, end)
}

// UserFundingHistory fetches the per-user funding-payment stream.
func (i *Info) UserFundingHistory(
	user string,
	startTime int64,
	endTime *int64,
) ([]UserFundingHistory, error) {
	resp, err := i.postTimeRangeRequest("userFunding", user, startTime, endTime, nil)
	if err != nil {
		return nil, err
	}

	var result []UserFundingHistory
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user funding history: %w", err)
	}
	return result, nil
}
