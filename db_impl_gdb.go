package sqlboiler

import (
	"context"
)

type Gdb interface {
	Db
}

type gdbImpl struct {
	*dbImpl
}

func NewGdb(ctx context.Context) Gdb {
	return &gdbImpl{
		dbImpl: &dbImpl{
			ctx:    ctx,
			copier: newDbCopier(),
		},
	}
}
