package sqlboiler

import (
	"context"

	"github.com/hdget/common/meta"
)

// Tdb Tenant db
type Tdb interface {
	Db
	Tid() int64 // 获取租户ID接口
}

type tdbImpl struct {
	*dbImpl
}

func NewTdb(ctx context.Context) Tdb {
	return &tdbImpl{
		dbImpl: &dbImpl{
			ctx:    ctx,
			copier: newDbCopier(),
		},
	}
}

func (impl *tdbImpl) Tid() int64 {
	return meta.FromServiceContext(impl.ctx).GetTid()
}
