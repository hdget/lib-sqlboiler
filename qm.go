package sqlboiler

import (
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/hdget/common/protobuf"
)

type qmBuilder struct {
	mods []qm.QueryMod
}

func NewQmBuilder(mods ...qm.QueryMod) *qmBuilder {
	return &qmBuilder{
		mods: mods,
	}
}

func (q *qmBuilder) Append(mods ...qm.QueryMod) *qmBuilder {
	q.mods = append(q.mods, mods...)
	return q
}

func (q *qmBuilder) Concat(modSlices ...[]qm.QueryMod) *qmBuilder {
	for _, mods := range modSlices {
		q.mods = append(q.mods, mods...)
	}
	return q
}

func (q *qmBuilder) Limit(list *protobuf.ListParam) *qmBuilder {
	p := getPaginator(list)
	q.mods = append(q.mods, qm.Offset(int(p.Offset)), qm.Limit(int(p.PageSize)))
	return q
}

func (q *qmBuilder) Output() []qm.QueryMod {
	return q.mods
}
