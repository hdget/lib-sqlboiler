package sqlboiler

import (
	"context"

	"github.com/aarondl/sqlboiler/v4/boil"
)

// ctxKeyTx db transaction
type ctxKeyTx struct{}

func ctxAddTx(ctx context.Context, value any) context.Context {
	return context.WithValue(ctx, ctxKeyTx{}, value)
}

func ctxGetTx(ctx context.Context) (boil.Transactor, bool) {
	t, ok := ctx.Value(ctxKeyTx{}).(boil.Transactor)
	if ok {
		return t, true
	}
	return nil, false
}
