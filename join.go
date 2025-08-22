package sqlboiler

import (
	"fmt"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"strings"
)

type joinKind int

const (
	joinKindUnknown joinKind = iota
	joinKindInner
	joinKindLeft
	joinKindRight
)

type JoinClauseBuilder struct {
	kind      joinKind
	joinTable string
	asTable   string
	clauses   []string
	quote     string
}

func innerJoin(quote, joinTable string, args ...string) *JoinClauseBuilder {
	var asTable string
	if len(args) > 0 {
		asTable = escape(args[0], quote, true)
	}
	return &JoinClauseBuilder{
		kind:      joinKindInner,
		joinTable: escape(joinTable, quote, true),
		asTable:   asTable,
		clauses:   make([]string, 0),
		quote:     quote,
	}
}

func leftJoin(quote, joinTable string, args ...string) *JoinClauseBuilder {
	var asTable string
	if len(args) > 0 {
		asTable = escape(args[0], quote, true)
	}
	return &JoinClauseBuilder{
		kind:      joinKindLeft,
		joinTable: escape(joinTable, quote, true),
		asTable:   asTable,
		clauses:   make([]string, 0),
		quote:     quote,
	}
}

func (j *JoinClauseBuilder) On(columnOrTableColumn, thatTableColumn string) *JoinClauseBuilder {
	leftColumn := escape(columnOrTableColumn, j.quote, true)
	rightColumn := escape(thatTableColumn, j.quote, true)
	if j.asTable != "" {
		j.clauses = append(j.clauses, fmt.Sprintf("%s AS %s ON %s.%s=%s", j.joinTable, j.asTable, j.asTable, leftColumn, rightColumn))
	} else {
		j.clauses = append(j.clauses, fmt.Sprintf("%s ON %s=%s", j.joinTable, leftColumn, rightColumn))
	}
	return j
}

func (j *JoinClauseBuilder) And(clause string) *JoinClauseBuilder {
	j.clauses = append(j.clauses, clause)
	return j
}

func (j *JoinClauseBuilder) Output(args ...any) qm.QueryMod {
	switch j.kind {
	case joinKindInner:
		return qm.InnerJoin(strings.Join(j.clauses, " AND "), args...)
	case joinKindLeft:
		return qm.LeftOuterJoin(strings.Join(j.clauses, " AND "), args...)
	case joinKindRight:
		return qm.RightOuterJoin(strings.Join(j.clauses, " AND "), args...)
	}
	return nil
}
