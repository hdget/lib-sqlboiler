package sqlboiler

import (
	"context"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/hdget/common/biz"
	"github.com/hdget/common/types"
	loggerUtils "github.com/hdget/utils/logger"
)

type Transactor interface {
	Finalize(err error)
	Executor() types.DbExecutor
}

type trans struct {
	tx     boil.Transactor
	errLog func(msg string, kvs ...any)
}

func NewTransactor(ctx biz.Context, logger types.LoggerProvider) (Transactor, error) {
	errLog := loggerUtils.Error
	if logger == nil {
		errLog = logger.Error
	}

	if v, ok := ctx.Value(biz.ContextKeyDbTransaction); ok {
		if tx, ok := v.(boil.Transactor); ok {
			return &trans{tx: tx, errLog: errLog}, nil
		}
	}

	// 没找到，则new
	tx, err := boil.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	// ctx保存transaction
	_ = ctx.WithValue(biz.ContextKeyDbTransaction, tx)

	return &trans{tx: tx, errLog: errLog}, nil
}

func (t *trans) Executor() types.DbExecutor {
	return t.tx
}

func (t *trans) Finalize(err error) {
	// need commit
	if err != nil {
		e := t.tx.Rollback()
		t.errLog("db roll back", "err", err, "rollback", e)
		return
	}

	e := t.tx.Commit()
	if e != nil {
		t.errLog("db commit", "err", e)
	}
}
