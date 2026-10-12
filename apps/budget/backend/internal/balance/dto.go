package balance

import "budget_me/backend/internal/balance/queries"

// balanceResponse: `set` false means the user has never stated a balance, and
// `balanceMinor` carries no meaning — the app offers to set one instead of
// showing a zero it made up.
type balanceResponse struct {
	Set          bool  `json:"set"`
	BalanceMinor int64 `json:"balanceMinor"`
}

type setBalanceRequest struct {
	BalanceMinor int64 `json:"balanceMinor"`
}

func toBalanceResponse(b queries.Balance) balanceResponse {
	return balanceResponse{Set: b.Set, BalanceMinor: b.BalanceMinor}
}
