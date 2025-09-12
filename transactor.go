package sqlboiler

import (
	"context"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/hdget/common/types"
	loggerUtils "github.com/hdget/utils/logger"
)

type Transactor interface {
	Finalize(err error)
	Context() context.Context
}

type trans struct {
	ctx    context.Context
	errLog func(msg string, kvs ...any)
}

func NewTransactor(ctx context.Context, logger types.LoggerProvider) (Transactor, error) {
	errLog := loggerUtils.Error
	if logger == nil {
		errLog = logger.Error
	}

	if _, ok := ctx.Value(types.CtxKeyTx{}).(boil.Transactor); ok {
		return &trans{ctx: ctx, errLog: errLog}, nil
	}

	// 没找到，则new
	tx, err := boil.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	return &trans{ctx: context.WithValue(ctx, types.CtxKeyTx{}, tx), errLog: errLog}, nil
}

func (t *trans) Context() context.Context {
	return t.ctx
}

func (t *trans) Finalize(err error) {
	tx := t.getTx()
	if tx == nil {
		return
	}

	// need commit
	if err != nil {
		e := tx.Rollback()
		t.errLog("db roll back", "err", err, "rollback", e)
		return
	}

	e := tx.Commit()
	if e != nil {
		t.errLog("db commit", "err", e)
	}
}

func (t *trans) getTx() boil.Transactor {
	if tx, ok := t.ctx.Value(types.CtxKeyTx{}).(boil.Transactor); ok {
		return tx
	}
	return nil
}
