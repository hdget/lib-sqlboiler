package sqlboiler

import (
	"context"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/hdget/common/types"
)

type Db interface {
	Copier() DbCopier
	Executor() boil.Executor
}

type dbImpl struct {
	ctx    context.Context
	copier DbCopier
}

func (impl *dbImpl) Executor() boil.Executor {
	if tx, ok := impl.ctx.Value(types.CtxKeyTx{}).(boil.Transactor); ok {
		return tx
	}
	return boil.GetDB()
}

func (impl *dbImpl) Copier() DbCopier {
	return newDbCopier()
}
