package sqlboiler

import (
	"context"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/hdget/common/types"
	loggerUtils "github.com/hdget/utils/logger"
)

type Transactor interface {
	Executor() boil.Executor
	Finalize(err error)
}

type trans struct {
	tx     boil.Transactor
	errLog func(msg string, kvs ...any)
}

func NewTransactor(logger types.LoggerProvider) (Transactor, error) {
	tx, err := boil.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	errLog := loggerUtils.Error
	if logger == nil {
		errLog = logger.Error
	}

	return &trans{tx: tx, errLog: errLog}, nil
}

func (t *trans) Executor() boil.Executor {
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
