package sqlboiler

import (
	"context"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/hdget/common/servicectx"
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
	if tx, ok := servicectx.GetTx(impl.ctx).(boil.Transactor); ok {
		return tx
	}
	return boil.GetDB()
}

func (impl *dbImpl) Copier() DbCopier {
	return newDbCopier()
}
