package sqlboiler

import (
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/hdget/common/biz"
)

type Db interface {
	Copier() DbCopier
	Executor() boil.Executor
}

type dbImpl struct {
	ctx    biz.Context
	copier DbCopier
}

func (impl *dbImpl) Executor() boil.Executor {
	if tx, ok := impl.ctx.Transactor().Get().(boil.Transactor); ok {
		return tx
	}
	return boil.GetDB()
}

func (impl *dbImpl) Copier() DbCopier {
	return newDbCopier()
}
