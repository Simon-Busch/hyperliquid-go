package hyperliquid

// Fills returns all fills for addr.
func (i *Info) Fills(addr string) ([]Fill, error) {
	return i.UserFills(addr)
}

// FillsBetween returns fills for addr in the range [start, end].
func (i *Info) FillsBetween(addr string, start int64, end *int64) ([]Fill, error) {
	return i.UserFillsByTime(addr, start, end)
}

// Order returns the order with the supplied exchange oid.
func (i *Info) Order(addr string, oid int64) (*OrderStatusResponse, error) {
	return i.QueryOrderByOid(addr, oid)
}

// OrderByCloid returns the order with the supplied client order id.
func (i *Info) OrderByCloid(addr, cloid string) (*OpenOrder, error) {
	return i.QueryOrderByCloid(addr, cloid)
}

// Fill returns the fill matching addr and oid.
func (i *Info) Fill(addr string, oid int64) (*Fill, error) {
	return i.QueryFillByOid(addr, oid)
}

// SubAccounts returns the sub-account list for addr.
func (i *Info) SubAccounts(addr string) ([]SubAccount, error) {
	return i.QuerySubAccounts(addr)
}

// Referral returns the referral state for addr.
func (i *Info) Referral(addr string) (*ReferralState, error) {
	return i.QueryReferralState(addr)
}

// MultiSigSigners returns the signer list for a multi-sig user.
func (i *Info) MultiSigSigners(multiSigAddr string) ([]MultiSigSigner, error) {
	return i.QueryUserToMultiSigSigners(multiSigAddr)
}

// InfoStakeGroup exposes the staking-info shortcuts. Accessed via the
// Info.Stake field, populated by NewInfo.
type InfoStakeGroup struct{ i *Info }

// Summary returns the staking summary for addr.
func (g *InfoStakeGroup) Summary(addr string) (*StakingSummary, error) {
	return g.i.UserStakingSummary(addr)
}

// Delegations returns the active delegations for addr.
func (g *InfoStakeGroup) Delegations(addr string) ([]StakingDelegation, error) {
	return g.i.UserStakingDelegations(addr)
}

// Rewards returns the staking reward history for addr.
func (g *InfoStakeGroup) Rewards(addr string) ([]StakingReward, error) {
	return g.i.UserStakingRewards(addr)
}
