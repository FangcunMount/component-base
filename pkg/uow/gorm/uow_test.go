package gormuow

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	gmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestWithinTransactionNilDBFailsClosed(t *testing.T) {
	t.Parallel()

	called := false
	err := NewUnitOfWork(nil).WithinTransaction(context.Background(), func(context.Context) error {
		called = true
		return nil
	})
	if !errors.Is(err, ErrUnitOfWorkUnavailable) {
		t.Fatalf("WithinTransaction() error = %v, want ErrUnitOfWorkUnavailable", err)
	}
	if called {
		t.Fatal("callback was called without a database")
	}
}

func TestWithTxAllowsRequireTxAndWithContext(t *testing.T) {
	t.Parallel()

	tx := &gorm.DB{}
	ctx := WithTx(context.Background(), tx)
	got, err := RequireTx(ctx)
	if err != nil {
		t.Fatalf("RequireTx() error = %v", err)
	}
	if got != tx {
		t.Fatal("RequireTx() did not return context transaction")
	}
}

func TestAfterCommitRequiresTransactionContext(t *testing.T) {
	t.Parallel()

	err := AfterCommit(context.Background(), func(context.Context) error { return nil })
	if !errors.Is(err, ErrActiveTransactionRequired) {
		t.Fatalf("AfterCommit() error = %v, want ErrActiveTransactionRequired", err)
	}
}

func TestWithinTransactionCommitRunsAfterCommit(t *testing.T) {
	t.Parallel()

	db, mock := newMockGORM(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	afterCommitCalled := false
	err := NewUnitOfWork(db).WithinTransaction(context.Background(), func(txCtx context.Context) error {
		if _, ok := TxFromContext(txCtx); !ok {
			t.Fatal("transaction missing from callback context")
		}
		return AfterCommit(txCtx, func(context.Context) error {
			afterCommitCalled = true
			return nil
		})
	})
	if err != nil {
		t.Fatalf("WithinTransaction() error = %v", err)
	}
	if !afterCommitCalled {
		t.Fatal("after commit hook was not called")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestWithinTransactionRollbackSkipsAfterCommit(t *testing.T) {
	t.Parallel()

	db, mock := newMockGORM(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	wantErr := errors.New("boom")
	afterCommitCalled := false
	err := NewUnitOfWork(db).WithinTransaction(context.Background(), func(txCtx context.Context) error {
		if err := AfterCommit(txCtx, func(context.Context) error {
			afterCommitCalled = true
			return nil
		}); err != nil {
			t.Fatalf("AfterCommit() error = %v", err)
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("WithinTransaction() error = %v, want %v", err, wantErr)
	}
	if afterCommitCalled {
		t.Fatal("after commit hook ran after rollback")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestWithinTransactionReusesExistingTransactionContext(t *testing.T) {
	t.Parallel()

	db, mock := newMockGORM(t)
	tx := &gorm.DB{}
	ctx := WithTx(context.Background(), tx)

	called := false
	err := NewUnitOfWork(db).WithinTransaction(ctx, func(txCtx context.Context) error {
		called = true
		got, ok := TxFromContext(txCtx)
		if !ok || got != tx {
			t.Fatal("existing transaction context was not reused")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithinTransaction() error = %v", err)
	}
	if !called {
		t.Fatal("callback was not called")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func newMockGORM(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(gmysql.New(gmysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return db, mock
}
