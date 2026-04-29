package gormuow

import (
	"context"
	"database/sql"
	"errors"

	"github.com/FangcunMount/component-base/pkg/log"
	"gorm.io/gorm"
)

var (
	ErrUnitOfWorkUnavailable     = errors.New("gorm unit of work unavailable")
	ErrActiveTransactionRequired = errors.New("gorm active transaction required")
)

type TxOptions struct {
	Name      string
	ReadOnly  bool
	Isolation sql.IsolationLevel
}

type txContextKey struct{}

type txState struct {
	tx          *gorm.DB
	afterCommit []func(context.Context) error
}

type UnitOfWork struct {
	db *gorm.DB
}

func NewUnitOfWork(db *gorm.DB) *UnitOfWork {
	return &UnitOfWork{db: db}
}

func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if tx == nil {
		return ctx
	}
	return context.WithValue(ctx, txContextKey{}, &txState{tx: tx})
}

func TxFromContext(ctx context.Context) (*gorm.DB, bool) {
	state, ok := txStateFromContext(ctx)
	if !ok || state.tx == nil {
		return nil, false
	}
	return state.tx, true
}

func RequireTx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := TxFromContext(ctx)
	if !ok {
		return nil, ErrActiveTransactionRequired
	}
	return tx, nil
}

func WithContext(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := TxFromContext(ctx); ok {
		return tx.WithContext(ctx)
	}
	if db == nil {
		return nil
	}
	return db.WithContext(ctx)
}

func AfterCommit(ctx context.Context, hook func(context.Context) error) error {
	if hook == nil {
		return nil
	}
	state, ok := txStateFromContext(ctx)
	if !ok || state.tx == nil {
		return ErrActiveTransactionRequired
	}
	state.afterCommit = append(state.afterCommit, hook)
	return nil
}

func (u *UnitOfWork) WithinTransaction(ctx context.Context, fn func(txCtx context.Context) error, opts ...TxOptions) error {
	if fn == nil {
		return nil
	}
	if u == nil || u.db == nil {
		return ErrUnitOfWorkUnavailable
	}
	if _, ok := TxFromContext(ctx); ok {
		return fn(ctx)
	}

	state := &txState{}
	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		state.tx = tx
		return fn(context.WithValue(ctx, txContextKey{}, state))
	}, toSQLTxOptions(opts...))
	if err != nil {
		return err
	}
	for _, hook := range state.afterCommit {
		if hookErr := hook(ctx); hookErr != nil {
			log.Warnf("after commit hook failed: %v", hookErr)
		}
	}
	return nil
}

func txStateFromContext(ctx context.Context) (*txState, bool) {
	if ctx == nil {
		return nil, false
	}
	state, ok := ctx.Value(txContextKey{}).(*txState)
	return state, ok && state != nil
}

func toSQLTxOptions(opts ...TxOptions) *sql.TxOptions {
	if len(opts) == 0 {
		return nil
	}
	opt := opts[0]
	if !opt.ReadOnly && opt.Isolation == sql.LevelDefault {
		return nil
	}
	return &sql.TxOptions{
		Isolation: opt.Isolation,
		ReadOnly:  opt.ReadOnly,
	}
}
