package sqlboiler

import (
	"context"

	commonTypes "github.com/hdget/common/types"
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
	if tid, ok := impl.ctx.Value(commonTypes.CtxKeyTid{}).(int64); ok {
		return tid
	}
	return 0
}
