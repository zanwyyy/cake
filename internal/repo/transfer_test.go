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

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.PingContext(ctx)
	require.NoError(t, err, "Failed to connect to test database")

	return db
}

// Helper function để lấy balance
func getBalance(t *testing.T, db *sql.DB, userID string) int64 {
	var balance int64
	err := db.QueryRow(`SELECT balance FROM users WHERE id = $1`, userID).Scan(&balance)
	require.NoError(t, err)
	return balance
}

func TestInsertTransaction_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &PostgresTransferRepo{db: db, kafka: &MockKafka{}}

	from := "1"
	to := "2"
	amount := int64(200)

	// Lấy balance trước khi transfer
	fromBalanceBefore := getBalance(t, db, from)
	toBalanceBefore := getBalance(t, db, to)
	t.Logf("Before transfer: from=%d, to=%d", fromBalanceBefore, toBalanceBefore)

	// Thực hiện giao dịch
	err := repo.InsertTransaction(context.Background(), from, to, amount)
	require.NoError(t, err)

	// Lấy balance sau khi transfer
	fromBalanceAfter := getBalance(t, db, from)
	toBalanceAfter := getBalance(t, db, to)
	t.Logf("After transfer: from=%d, to=%d", fromBalanceAfter, toBalanceAfter)

	// Kiểm tra số dư thay đổi chính xác
	require.Equal(t, fromBalanceBefore-amount, fromBalanceAfter,
		"From balance mismatch: expected %d, got %d",
		fromBalanceBefore-amount, fromBalanceAfter)
	require.Equal(t, toBalanceBefore+amount, toBalanceAfter,
		"To balance mismatch: expected %d, got %d",
		toBalanceBefore+amount, toBalanceAfter)
}

func TestInsertTransaction_InsufficientBalance(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Reset test data

	repo := &PostgresTransferRepo{db: db, kafka: &MockKafka{}}

	from := "1"
	to := "2"
	amount := int64(2000000) // Lớn hơn balance

	// Lấy balance trước khi transfer
	fromBalanceBefore := getBalance(t, db, from)
	toBalanceBefore := getBalance(t, db, to)

	err := repo.InsertTransaction(context.Background(), from, to, amount)
	require.Error(t, err)

	// Verify balance không thay đổi
	fromBalanceAfter := getBalance(t, db, from)
	toBalanceAfter := getBalance(t, db, to)

	require.Equal(t, fromBalanceBefore, fromBalanceAfter)
	require.Equal(t, toBalanceBefore, toBalanceAfter)

	// Verify không có transaction record được tạo
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE from_user = $1 AND to_user = $2 AND amount = $3`,
		from, to, amount).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestInsertTransaction_Concurrent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Reset test data

	repo := &PostgresTransferRepo{db: db, kafka: &MockKafka{}}

	from := "1"
	to := "2"

	n := 1000
	amount := int64(50)

	// Lấy balance trước khi test
	fromBalanceBefore := getBalance(t, db, from)
	toBalanceBefore := getBalance(t, db, to)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var successCount int
	var errors []error

	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(index int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := repo.InsertTransaction(ctx, from, to, amount)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errors = append(errors, fmt.Errorf("goroutine %d: %w", index, err))
			} else {
				successCount++
			}
		}(i)
	}

	wg.Wait()

	// Log errors nếu có
	if len(errors) > 0 {
		t.Logf("Concurrent test had %d errors out of %d attempts:", len(errors), n)
		for _, err := range errors {
			t.Logf("  - %v", err)
		}
	}

	// Verify final balances
	fromBalanceAfter := getBalance(t, db, from)
	toBalanceAfter := getBalance(t, db, to)

	// Balance changes should match successful transactions
	expectedFromBalance := fromBalanceBefore - int64(successCount)*amount
	expectedToBalance := toBalanceBefore + int64(successCount)*amount

	require.Equal(t, expectedFromBalance, fromBalanceAfter,
		"From balance mismatch. Success: %d, Expected: %d, Actual: %d",
		successCount, expectedFromBalance, fromBalanceAfter)
	require.Equal(t, expectedToBalance, toBalanceAfter,
		"To balance mismatch. Success: %d, Expected: %d, Actual: %d",
		successCount, expectedToBalance, toBalanceAfter)

	// Verify transaction records
	var recordCount int
	err := db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE from_user = $1 AND to_user = $2 AND amount = $3`,
		from, to, amount).Scan(&recordCount)
	require.NoError(t, err)
	require.Equal(t, successCount, recordCount,
		"Transaction record count should match successful operations")

	// Trong concurrent test, có thể một số transactions thất bại do race conditions
	// Nhưng tổng balance phải consistent
	t.Logf("Concurrent test results: %d/%d transactions succeeded", successCount, n)

	// Verify tổng tiền trong hệ thống không đổi
	totalBefore := fromBalanceBefore + toBalanceBefore
	totalAfter := fromBalanceAfter + toBalanceAfter
	require.Equal(t, totalBefore, totalAfter, "Total money in system should remain constant")
}

func TestInsertTransaction_InvalidUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &PostgresTransferRepo{db: db, kafka: &MockKafka{}}

	// Test với user không tồn tại
	err := repo.InsertTransaction(context.Background(), "nonexistent", "1", 100)
	require.Error(t, err)

	err = repo.InsertTransaction(context.Background(), "1", "nonexistent", 100)
	require.Error(t, err)
}
