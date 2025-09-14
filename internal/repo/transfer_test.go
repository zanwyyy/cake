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
	amount := int64(2)

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

	repo := &PostgresTransferRepo{db: db, kafka: &MockKafka{}}

	from := "1"
	to := "2"
	amount := int64(2000000) // Lớn hơn balance

	// Lấy balance trước khi transfer
	fromBalanceBefore := getBalance(t, db, from)
	toBalanceBefore := getBalance(t, db, to)

	// Đếm số transaction trước khi chạy
	var countBefore int
	err := db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&countBefore)
	require.NoError(t, err)

	// Thực hiện giao dịch
	err = repo.InsertTransaction(context.Background(), from, to, amount)
	require.Error(t, err)

	// Verify balance không thay đổi
	fromBalanceAfter := getBalance(t, db, from)
	toBalanceAfter := getBalance(t, db, to)

	require.Equal(t, fromBalanceBefore, fromBalanceAfter)
	require.Equal(t, toBalanceBefore, toBalanceAfter)

	// Verify tổng số transaction không đổi
	var countAfter int
	err = db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&countAfter)
	require.NoError(t, err)
	require.Equal(t, countBefore, countAfter)
}

func TestInsertTransaction_WorkerPool(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &PostgresTransferRepo{db: db, kafka: &MockKafka{}}

	from := "1"
	to := "2"

	n := 200 // tổng số transaction
	amount := int64(1)
	workers := 50 // số worker đồng thời

	fromBalanceBefore := getBalance(t, db, from)
	toBalanceBefore := getBalance(t, db, to)

	// Channel để phát job
	jobs := make(chan int, n)
	// Channel để gom lỗi
	errs := make(chan error, n)

	var wg sync.WaitGroup

	// Tạo worker pool
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range jobs {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				err := repo.InsertTransaction(ctx, from, to, amount)
				cancel()

				if err != nil {
					errs <- fmt.Errorf("job %d by worker %d failed: %w", j, id, err)
				}
			}
		}(w)
	}

	// Bắn job vào pool
	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)

	// Đợi tất cả worker xong
	wg.Wait()
	close(errs)

	// Log lỗi nếu có
	for err := range errs {
		t.Log(err)
	}

	// Kiểm tra số dư cuối cùng
	fromBalanceAfter := getBalance(t, db, from)
	toBalanceAfter := getBalance(t, db, to)

	expectedFrom := fromBalanceBefore - int64(n)*amount
	expectedTo := toBalanceBefore + int64(n)*amount

	require.Equal(t, expectedFrom, fromBalanceAfter,
		"Số dư người gửi không đúng. Expected %d, got %d", expectedFrom, fromBalanceAfter)

	require.Equal(t, expectedTo, toBalanceAfter,
		"Số dư người nhận không đúng. Expected %d, got %d", expectedTo, toBalanceAfter)

	// Tổng tiền hệ thống không đổi
	totalBefore := fromBalanceBefore + toBalanceBefore
	totalAfter := fromBalanceAfter + toBalanceAfter
	require.Equal(t, totalBefore, totalAfter,
		"Tổng tiền trong hệ thống phải không đổi")

	t.Logf("From: %d -> %d | To: %d -> %d",
		fromBalanceBefore, fromBalanceAfter,
		toBalanceBefore, toBalanceAfter)
}

func TestInsertTransaction_InvalidUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &PostgresTransferRepo{db: db, kafka: &MockKafka{}}

	err := repo.InsertTransaction(context.Background(), "nonexistent", "1", 100)
	require.Error(t, err)

	err = repo.InsertTransaction(context.Background(), "1", "nonexistent", 100)
	require.Error(t, err)
}
