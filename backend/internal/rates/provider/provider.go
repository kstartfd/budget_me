package provider

import "context"

type Provider interface {
	Rate(ctx context.Context, currency string) (float64, error)
}
