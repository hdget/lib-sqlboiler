package sqlboiler

import (
	"context"
	"github.com/hdget/common/intf"
	loggerUtils "github.com/hdget/utils/logger"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

type Transaction struct {
	Tx         boil.Transactor
	needCommit bool
	errLog     func(msg string, kvs ...any)
}

func NewTransaction(logger intf.LoggerProvider, args ...boil.Transactor) (*Transaction, error) {
	var (
		err error
		tx  boil.Transactor
	)

	needCommit := true
	if len(args) > 0 && args[0] != nil {
		tx = args[0]
		// 外部传递过来的transactor我们不需要commit
		needCommit = false
	} else {
		tx, err = boil.BeginTx(context.Background(), nil)
	}
	if err != nil {
		return nil, err
	}

	errLog := loggerUtils.Error
	if logger == nil {
		errLog = logger.Error
	}

	return &Transaction{Tx: tx, needCommit: needCommit, errLog: errLog}, nil
}

func (t Transaction) CommitOrRollback(err error) {
	if !t.needCommit {
		return
	}

	// need commit
	if err != nil {
		e := t.Tx.Rollback()
		t.errLog("db roll back", "err", err, "rollback", e)
		return
	}

	e := t.Tx.Commit()
	if e != nil {
		t.errLog("db commit", "err", e)
	}
}
