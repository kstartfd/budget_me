package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Belarusbank struct {
	baseURL string
	http    *http.Client
}

func NewBelarusbank(baseURL string) *Belarusbank {
	return &Belarusbank{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (b *Belarusbank) Rate(ctx context.Context, currency string) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.baseURL+"/api/kursExchange", nil)
	if err != nil {
		return 0, err
	}

	res, err := b.http.Do(req)
	if err != nil {
		return 0, err
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("rates: belarusbank api status %d", res.StatusCode)
	}

	var branches []belarusbankBranch
	if err := json.NewDecoder(res.Body).Decode(&branches); err != nil {
		return 0, err
	}

	if len(branches) == 0 {
		return 0, errors.New("rates: belarusbank returned no branches")
	}

	var inRaw, outRaw string

	// scale is how many units the quote is for. The bank prices the Russian
	// ruble per 100 (RUB_in "3.10" means 3.10 Br for 100 ₽, not for one) — the
	// same convention NBRB uses via Cur_Scale.
	scale := 1.0

	switch strings.ToUpper(currency) {
	case "USD":
		inRaw, outRaw = branches[0].USDIn, branches[0].USDOut
	case "EUR":
		inRaw, outRaw = branches[0].EURIn, branches[0].EUROut
	case "RUB":
		inRaw, outRaw, scale = branches[0].RUBIn, branches[0].RUBOut, 100
	default:
		return 0, fmt.Errorf("rates: belarusbank unsupported currency %s", currency)
	}

	in, err1 := strconv.ParseFloat(inRaw, 64)
	out, err2 := strconv.ParseFloat(outRaw, 64)

	if err1 != nil || err2 != nil || in <= 0 || out <= 0 {
		return 0, fmt.Errorf("rates: belarusbank has no %s rate", strings.ToUpper(currency))
	}

	return (in + out) / 2 / scale, nil
}

type belarusbankBranch struct {
	USDIn  string `json:"USD_in"`
	USDOut string `json:"USD_out"`
	EURIn  string `json:"EUR_in"`
	EUROut string `json:"EUR_out"`
	RUBIn  string `json:"RUB_in"`
	RUBOut string `json:"RUB_out"`
}
