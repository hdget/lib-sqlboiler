package sqlboiler

import (
	"github.com/hdget/common/biz"
)

type Gdb interface {
	Db
}

type gdbImpl struct {
	*dbImpl
}

func NewGdb(ctx biz.Context) Gdb {
	return &gdbImpl{
		dbImpl: &dbImpl{
			ctx:    ctx,
			copier: newDbCopier(),
		},
	}
}
