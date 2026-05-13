package hyperliquid

import (
	"encoding/json"
	"fmt"
)

// Meta retrieves perpetuals metadata. If dex is provided and non-empty,
// the snapshot is pinned to that HIP-3 dex; otherwise the default dex is
// returned.
func (i *Info) Meta(dex ...string) (*Meta, error) {
	payload := map[string]any{
		"type": "meta",
	}
	if len(dex) > 0 && dex[0] != "" {
		payload["dex"] = dex[0]
	}

	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch meta: %w", err)
	}

	return parseMetaResponse(resp)
}

func parseMetaResponse(resp []byte) (*Meta, error) {
	var meta map[string]json.RawMessage
	if err := json.Unmarshal(resp, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal meta response: %w", err)
	}

	var universe []AssetInfo
	if err := json.Unmarshal(meta["universe"], &universe); err != nil {
		return nil, fmt.Errorf("failed to unmarshal universe: %w", err)
	}

	var collateralToken int
	if ct, ok := meta["collateralToken"]; ok {
		if err := json.Unmarshal(ct, &collateralToken); err != nil {
			return nil, fmt.Errorf("failed to unmarshal collateralToken: %w", err)
		}
	}

	var marginTables []MarginTable
	if mt, ok := meta["marginTables"]; ok {
		var marginTablesRaw [][]any
		if err := json.Unmarshal(mt, &marginTablesRaw); err != nil {
			return nil, fmt.Errorf("failed to unmarshal margin tables: %w", err)
		}

		marginTables = make([]MarginTable, len(marginTablesRaw))
		for idx, marginTable := range marginTablesRaw {
			if len(marginTable) < 2 {
				continue
			}
			id := int(marginTable[0].(float64))
			tableBytes, err := json.Marshal(marginTable[1])
			if err != nil {
				return nil, fmt.Errorf("failed to marshal margin table data: %w", err)
			}

			var marginTableData map[string]any
			if err := json.Unmarshal(tableBytes, &marginTableData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal margin table data: %w", err)
			}

			marginTiersBytes, err := json.Marshal(marginTableData["marginTiers"])
			if err != nil {
				return nil, fmt.Errorf("failed to marshal margin tiers: %w", err)
			}

			var marginTiers []MarginTier
			if err := json.Unmarshal(marginTiersBytes, &marginTiers); err != nil {
				return nil, fmt.Errorf("failed to unmarshal margin tiers: %w", err)
			}

			desc := ""
			if d, ok := marginTableData["description"].(string); ok {
				desc = d
			}

			marginTables[idx] = MarginTable{
				ID:          id,
				Description: desc,
				MarginTiers: marginTiers,
			}
		}
	}

	return &Meta{
		Universe:        universe,
		MarginTables:    marginTables,
		CollateralToken: collateralToken,
	}, nil
}

// SpotMeta retrieves the canonical spot metadata.
func (i *Info) SpotMeta() (*SpotMeta, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "spotMeta",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot meta: %w", err)
	}

	var spotMeta SpotMeta
	if err := json.Unmarshal(resp, &spotMeta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spot meta response: %w", err)
	}

	return &spotMeta, nil
}

// OutcomeMeta retrieves metadata for HIP-4 binary outcome (prediction)
// markets. Returns the list of active outcomes, each with its sides
// (YES/NO). May return an empty slice if no markets are live on the
// target environment.
func (i *Info) OutcomeMeta() (*OutcomeMeta, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "outcomeMeta",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch outcome meta: %w", err)
	}

	var outcomeMeta OutcomeMeta
	if err := json.Unmarshal(resp, &outcomeMeta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal outcome meta response: %w", err)
	}

	return &outcomeMeta, nil
}

// PerpDexs returns the list of available perpetual dexes. Each element is
// either nil (the default dex) or a PerpDex object. The first element is
// always nil, representing the default dex.
func (i *Info) PerpDexs() (MixedArray, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "perpDexs",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch perp dexs: %w", err)
	}

	var result MixedArray
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal perp dexs: %w", err)
	}
	return result, nil
}
