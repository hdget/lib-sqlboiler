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
	ctx    biz.Context
	errLog func(msg string, kvs ...any)
}

func NewTransactor(ctx biz.Context, logger types.LoggerProvider) (Transactor, error) {
	errLog := loggerUtils.Error
	if logger != nil {
		errLog = logger.Error
	}

	var err error
	var transactor boil.Transactor
	if v, ok := ctx.Transactor().Get().(boil.Transactor); ok {
		transactor = v
	} else { // 没找到，则new
		transactor, err = boil.BeginTx(context.Background(), nil)
		if err != nil {
			return nil, err
		}
	}

	// ctx保存transaction
	ctx.Transactor().Ref(transactor)

	return &trans{tx: transactor, ctx: ctx, errLog: errLog}, nil
}

func (t *trans) Finalize(err error) {
	if needFinalize := t.ctx.Transactor().Unref(); !needFinalize {
		return
	}

	// transaction执行完以后需要从ctx中移除
	defer func() {
		t.ctx.Transactor().Destroy()
	}()
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
