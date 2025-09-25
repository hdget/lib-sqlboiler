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

	if tx, ok := ctx.GetTx().(boil.Transactor); ok {
		return &trans{tx: tx, errLog: errLog}, nil
	}

	// 没找到，则new
	tx, err := boil.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	return &trans{tx: tx, errLog: errLog}, nil
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
