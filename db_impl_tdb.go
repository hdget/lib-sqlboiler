package sqlboiler

import (
	"github.com/hdget/common/biz"
)

// Tdb Tenant db
type Tdb interface {
	Db
	Tid() int64 // 获取租户ID接口, 本来可以从ctx中获取，但为了区分Gdb和Tdb，强制实现冗余接口
}

type tdbImpl struct {
	*dbImpl
}

func NewTdb(ctx biz.Context) Tdb {
	return &tdbImpl{
		dbImpl: &dbImpl{
			ctx:    ctx,
			copier: newDbCopier(),
		},
	}
}

func (impl *tdbImpl) Tid() int64 {
	return impl.ctx.Tid()
}
