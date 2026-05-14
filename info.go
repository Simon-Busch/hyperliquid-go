package hyperliquid

import (
	"fmt"
)

const (
	// spotAssetIndexOffset is the offset added to spot asset indices.
	spotAssetIndexOffset = 10000
	// builderPerpAssetBase is the base offset for builder-deployed perp
	// asset ids. See the Asset IDs docs:
	// asset = 100000 + perpDexIndex*10000 + indexInMeta.
	builderPerpAssetBase = 100000
	// outcomeAssetBase is the base offset for HIP-4 outcome (binary
	// prediction) market asset ids. See Asset IDs docs:
	// asset = 100_000_000 + (10*outcome + side).
	outcomeAssetBase = 100000000
)

// Info is the read-only query surface. Construct it indirectly via New.
type Info struct {
	client         *httpAPI
	coinToAsset    map[string]int
	nameToCoin     map[string]string
	assetToDecimal map[int]int
	perpDexName    string // For HIP-3 builder-deployed perps (e.g., "flx")

	// Stake exposes the staking-info sub-group.
	Stake *InfoStakeGroup
}

// postTimeRangeRequest makes a POST request with time-range parameters.
func (i *Info) postTimeRangeRequest(
	requestType, user string,
	startTime int64,
	endTime *int64,
	extraParams map[string]any,
) ([]byte, error) {
	payload := map[string]any{
		"type":      requestType,
		"startTime": startTime,
	}
	if user != "" {
		payload["user"] = user
	}
	if endTime != nil {
		payload["endTime"] = *endTime
	}
	for k, v := range extraParams {
		payload[k] = v
	}

	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", requestType, err)
	}
	return resp, nil
}

// NewInfo creates a new Info instance. perpDexName is optional — pass an
// empty string for the default perp dex, or a builder dex name (e.g.
// "flx") for HIP-3 builder-deployed perps.
func NewInfo(baseURL string, skipWS bool, meta *Meta, spotMeta *SpotMeta, perpDexs *MixedArray, perpDexName string) *Info {
	info := &Info{
		client:         newHTTPAPI(baseURL, nil),
		coinToAsset:    make(map[string]int),
		nameToCoin:     make(map[string]string),
		assetToDecimal: make(map[int]int),
		perpDexName:    perpDexName,
	}
	info.Stake = &InfoStakeGroup{i: info}

	if meta == nil {
		var err error
		// When the client is pinned to a HIP-3 builder dex, the asset
		// registration below iterates the meta universe under the
		// assumption that it lists coins for THAT dex. Fetching the
		// default-dex meta would silently register the wrong universe
		// and break every subsequent AssetID / validate lookup.
		if info.perpDexName != "" {
			meta, err = info.Meta(info.perpDexName)
		} else {
			meta, err = info.Meta()
		}
		if err != nil {
			panic(err)
		}
	}

	if spotMeta == nil {
		var err error
		spotMeta, err = info.SpotMeta()
		if err != nil {
			panic(err)
		}
	}

	// Map perp assets
	if info.perpDexName != "" {
		// Builder-deployed perp: compute full asset id as documented.
		if perpDexs == nil {
			var err error
			perpDexsNew, err := info.PerpDexs()
			perpDexs = &perpDexsNew
			if err != nil {
				panic(err)
			}
		}
		perpDexIndex := -1
		for i, mv := range *perpDexs {
			if mv.Type() != "object" {
				continue
			}
			var pd PerpDex
			if err := mv.Parse(&pd); err == nil && pd.Name == info.perpDexName {
				perpDexIndex = i
				break
			}
		}
		if perpDexIndex < 0 {
			panic(fmt.Errorf("unknown perp dex %q (not present in /info perpDexs)", info.perpDexName))
		}
		base := builderPerpAssetBase + perpDexIndex*10000
		for idxInMeta, assetInfo := range meta.Universe {
			assetID := base + idxInMeta
			info.coinToAsset[assetInfo.Name] = assetID
			info.nameToCoin[assetInfo.Name] = assetInfo.Name
			info.assetToDecimal[assetID] = assetInfo.SzDecimals
		}
	} else {
		// Default perp dex: asset id is just index in meta universe.
		for asset, assetInfo := range meta.Universe {
			info.coinToAsset[assetInfo.Name] = asset
			info.nameToCoin[assetInfo.Name] = assetInfo.Name
			info.assetToDecimal[asset] = assetInfo.SzDecimals
		}
	}

	// Map spot assets starting at 10000. spotInfo.Tokens[0] is the index
	// of the base token in spotMeta.Tokens; Hyperliquid occasionally
	// returns spot entries whose base-token index is past the end of the
	// Tokens array (placeholder slots / paginated tail). In that case we
	// register the asset id but skip the szDecimals lookup rather than
	// panic — callers that need the decimals will get 0 and can fall
	// back to a metadata refresh.
	for _, spotInfo := range spotMeta.Universe {
		asset := spotInfo.Index + spotAssetIndexOffset
		info.coinToAsset[spotInfo.Name] = asset
		info.nameToCoin[spotInfo.Name] = spotInfo.Name
		if len(spotInfo.Tokens) > 0 {
			baseIdx := spotInfo.Tokens[0]
			if baseIdx >= 0 && baseIdx < len(spotMeta.Tokens) {
				info.assetToDecimal[asset] = spotMeta.Tokens[baseIdx].SzDecimals
			}
		}
	}

	// Map HIP-4 outcome assets starting at 100_000_000.
	// Failure here is non-fatal: outcomeMeta may be empty or missing on
	// some environments, and the SDK should still work for perp/spot users.
	if outcomeMeta, err := info.OutcomeMeta(); err == nil {
		for _, oc := range outcomeMeta.Outcomes {
			for sideIdx, spec := range oc.SideSpecs {
				enc := 10*oc.Outcome + sideIdx
				asset := outcomeAssetBase + enc
				// Canonical name is "#<enc>" — empirically verified as the
				// form the exchange echoes in l2Book/allMids responses.
				// "+<enc>" form does NOT work for L2 queries despite being
				// mentioned in the docs, so we don't register it.
				canonical := fmt.Sprintf("#%d", enc)
				friendly := fmt.Sprintf("%s:%s", oc.Name, spec.Name)
				info.coinToAsset[canonical] = asset
				info.coinToAsset[friendly] = asset
				info.nameToCoin[canonical] = canonical
				info.nameToCoin[friendly] = canonical
				// szDecimals=0: HIP-4 contracts are integer-quantised. The L2
				// wire format shows sizes like "18.0" but the exchange rejects
				// fractional sizes ("Order has invalid size"). outcomeMeta
				// does not expose szDecimals; revisit if exchange behavior
				// changes.
				info.assetToDecimal[asset] = 0
			}
		}
	}

	return info
}

// PerpDexName returns the configured builder perp dex name (e.g. "flx"),
// or empty string for the default dex.
func (i *Info) PerpDexName() string {
	return i.perpDexName
}
