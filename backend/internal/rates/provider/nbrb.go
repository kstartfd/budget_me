package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type NBRB struct {
	baseURL string
	http    *http.Client
}

func NewNBRB(baseURL string) *NBRB {
	return &NBRB{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (n *NBRB) Rate(ctx context.Context, currency string) (float64, error) {
	url := fmt.Sprintf("%s/exrates/rates/%s?parammode=2", n.baseURL, currency)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	res, err := n.http.Do(req)
	if err != nil {
		return 0, err
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("rates: nbrb api status %d", res.StatusCode)
	}

	var parsed nbrbRate
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return 0, err
	}

	if parsed.Scale <= 0 || parsed.OfficialRate <= 0 {
		return 0, fmt.Errorf("rates: nbrb returned no rate for %s", currency)
	}

	return parsed.OfficialRate / float64(parsed.Scale), nil
}

type nbrbRate struct {
	Abbreviation string  `json:"Cur_Abbreviation"`
	Scale        int     `json:"Cur_Scale"`
	OfficialRate float64 `json:"Cur_OfficialRate"`
}
