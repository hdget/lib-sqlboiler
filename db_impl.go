package sqlboiler

import (
	"context"
	"github.com/aarondl/sqlboiler/v4/boil"
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
	if tx, ok := ctxGetTx(impl.ctx); ok {
		return tx
	}
	return boil.GetDB()
}

func (impl *dbImpl) Copier() DbCopier {
	return newDbCopier()
}
