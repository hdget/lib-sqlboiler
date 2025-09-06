package sqlboiler

import (
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/hdget/common/protobuf"
	"slices"
)

type QmBuilder interface {
	Append(mods ...qm.QueryMod) QmBuilder
	Concat(modSlices []qm.QueryMod) QmBuilder
	Limit(list ...*protobuf.ListParam) QmBuilder
	Output() []qm.QueryMod
}

type qmBuilderImpl struct {
	mods []qm.QueryMod
}

func NewQmBuilder(mods ...qm.QueryMod) QmBuilder {
	return &qmBuilderImpl{
		mods: mods,
	}
}

func (q *qmBuilderImpl) Append(mods ...qm.QueryMod) QmBuilder {
	if len(mods) > 0 {
		q.mods = slices.Concat(q.mods, mods)
	}
	return q
}

func (q *qmBuilderImpl) Concat(modSlices []qm.QueryMod) QmBuilder {
	if len(modSlices) > 0 {
		q.mods = slices.Concat(q.mods, modSlices)
	}
	return q
}

func (q *qmBuilderImpl) Limit(list ...*protobuf.ListParam) QmBuilder {
	if len(list) == 0 {
		return q
	}

	p := getPaginator(list[0])
	q.mods = append(q.mods, qm.Offset(int(p.Offset)), qm.Limit(int(p.PageSize)))
	return q
}

func (q *qmBuilderImpl) Output() []qm.QueryMod {
	return q.mods
}
