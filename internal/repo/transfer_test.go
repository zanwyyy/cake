package repo

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type MockKafka struct{}

func (m *MockKafka) Publish(ctx context.Context, key, msg string) error {
	log.Printf("[MockKafka] publish: %s", msg)
	return nil
}

func setupTestDB(t *testing.T) *sql.DB {
	connStr := "postgres://demo_user:demo_pass@localhost:5432/demo_db?sslmode=disable&connect_timeout=10"

	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.PingContext(ctx)
	require.NoError(t, err, "Failed to connect to test database")

	return db
}

func TestInsertTransaction_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := &PostgresTransferRepo{db: db, kafka: &MockKafka{}}

	from := "1"
	to := "2"
	amount := int64(2)

	fromBalanceBefore, err := repo.GetBalance(ctx, from)
	require.NoError(t, err)
	toBalanceBefore, err := repo.GetBalance(ctx, to)
	require.NoError(t, err)

	t.Logf("Trước transfer: from=%d, to=%d", fromBalanceBefore, toBalanceBefore)

	err = repo.InsertTransaction(ctx, from, to, amount)
	require.NoError(t, err)

	fromBalanceAfter, err := repo.GetBalance(ctx, from)
	require.NoError(t, err)
	toBalanceAfter, err := repo.GetBalance(ctx, to)
	require.NoError(t, err)

	t.Logf("Sau transfer: from=%d, to=%d", fromBalanceAfter, toBalanceAfter)

	require.Equal(t, fromBalanceBefore-amount, fromBalanceAfter)
	require.Equal(t, toBalanceBefore+amount, toBalanceAfter)
}

func TestInsertTransaction_InsufficientBalance(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := &PostgresTransferRepo{db: db, kafka: &MockKafka{}}

	from := "1"
	to := "2"
	amount := int64(2000000)

	fromBalanceBefore, err := repo.GetBalance(ctx, from)
	require.NoError(t, err)
	toBalanceBefore, err := repo.GetBalance(ctx, to)
	require.NoError(t, err)

	var countBefore int
	err = db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&countBefore)
	require.NoError(t, err)

	err = repo.InsertTransaction(ctx, from, to, amount)
	require.Error(t, err)

	fromBalanceAfter, err := repo.GetBalance(ctx, from)
	require.NoError(t, err)
	toBalanceAfter, err := repo.GetBalance(ctx, to)
	require.NoError(t, err)

	require.Equal(t, fromBalanceBefore, fromBalanceAfter)
	require.Equal(t, toBalanceBefore, toBalanceAfter)

	var countAfter int
	err = db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&countAfter)
	require.NoError(t, err)
	require.Equal(t, countBefore, countAfter)
}

func TestInsertTransaction_Concurrent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &PostgresTransferRepo{db: db, kafka: &MockKafka{}}

	from := "1"
	to := "2"
	n := 50
	amount := int64(1)

	ctx := context.Background()
	fromBalanceBefore, err := repo.GetBalance(ctx, from)
	require.NoError(t, err)
	toBalanceBefore, err := repo.GetBalance(ctx, to)
	require.NoError(t, err)

	var wg sync.WaitGroup
	errs := make(chan error, n)

	startLine := make(chan struct{})

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(job int) {
			defer wg.Done()
			<-startLine

			jobCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := repo.InsertTransaction(jobCtx, from, to, amount); err != nil {
				errs <- fmt.Errorf("job %d failed: %w", job, err)
			}
		}(i)
	}

	close(startLine)

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Log(err)
	}

	fromBalanceAfter, err := repo.GetBalance(ctx, from)
	require.NoError(t, err)
	toBalanceAfter, err := repo.GetBalance(ctx, to)
	require.NoError(t, err)

	expectedFrom := fromBalanceBefore - int64(n)*amount
	expectedTo := toBalanceBefore + int64(n)*amount

	require.Equal(t, expectedFrom, fromBalanceAfter, "Số dư người gửi không đúng")
	require.Equal(t, expectedTo, toBalanceAfter, "Số dư người nhận không đúng")

	totalBefore := fromBalanceBefore + toBalanceBefore
	totalAfter := fromBalanceAfter + toBalanceAfter
	require.Equal(t, totalBefore, totalAfter, "Tổng tiền trong hệ thống phải không đổi")

	t.Logf("From: %d -> %d | To: %d -> %d",
		fromBalanceBefore, fromBalanceAfter,
		toBalanceBefore, toBalanceAfter)
}

func TestInsertTransaction_InvalidUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := &PostgresTransferRepo{db: db, kafka: &MockKafka{}}

	err := repo.InsertTransaction(ctx, "nonexistent", "1", 100)
	require.Error(t, err)

	err = repo.InsertTransaction(ctx, "1", "nonexistent", 100)
	require.Error(t, err)
}
