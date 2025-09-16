package repo

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setup test DB with GORM
func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "postgres://demo_user:demo_pass@localhost:5432/demo_db?sslmode=disable&connect_timeout=10"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = sqlDB.PingContext(ctx)
	require.NoError(t, err, "Failed to connect to test database")

	return db
}

type MockPubSub struct{}

func (m *MockPubSub) Publish(msg []byte) error {
	return nil
}

func (m *MockPubSub) Subscribe(ctx context.Context) error {
	return nil
}

func TestInsertTransaction_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormTransferRepo{db: db, pubsub: &MockPubSub{}}

	ctx := context.Background()
	from := "1"
	to := "2"
	amount := int64(2)

	var fromBalanceBefore, toBalanceBefore int64
	require.NoError(t, db.Table("users").Select("balance").Where("id = ?", from).Scan(&fromBalanceBefore).Error)
	require.NoError(t, db.Table("users").Select("balance").Where("id = ?", to).Scan(&toBalanceBefore).Error)

	err := repo.InsertTransaction(ctx, from, to, amount)
	require.NoError(t, err)

	var fromBalanceAfter, toBalanceAfter int64
	require.NoError(t, db.Table("users").Select("balance").Where("id = ?", from).Scan(&fromBalanceAfter).Error)
	require.NoError(t, db.Table("users").Select("balance").Where("id = ?", to).Scan(&toBalanceAfter).Error)

	require.Equal(t, fromBalanceBefore-amount, fromBalanceAfter)
	require.Equal(t, toBalanceBefore+amount, toBalanceAfter)
}

func TestInsertTransaction_InsufficientBalance(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormTransferRepo{db: db, pubsub: &MockPubSub{}}

	ctx := context.Background()
	from := "1"
	to := "2"
	amount := int64(2000000)

	var fromBalanceBefore, toBalanceBefore int64
	require.NoError(t, db.Table("users").Select("balance").Where("id = ?", from).Scan(&fromBalanceBefore).Error)
	require.NoError(t, db.Table("users").Select("balance").Where("id = ?", to).Scan(&toBalanceBefore).Error)

	var countBefore int64
	require.NoError(t, db.Table("transactions").Count(&countBefore).Error)

	err := repo.InsertTransaction(ctx, from, to, amount)
	require.Error(t, err)

	var fromBalanceAfter, toBalanceAfter int64
	require.NoError(t, db.Table("users").Select("balance").Where("id = ?", from).Scan(&fromBalanceAfter).Error)
	require.NoError(t, db.Table("users").Select("balance").Where("id = ?", to).Scan(&toBalanceAfter).Error)

	require.Equal(t, fromBalanceBefore, fromBalanceAfter)
	require.Equal(t, toBalanceBefore, toBalanceAfter)

	var countAfter int64
	require.NoError(t, db.Table("transactions").Count(&countAfter).Error)
	require.Equal(t, countBefore, countAfter)
}

func TestInsertTransaction_Concurrent(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormTransferRepo{db: db, pubsub: &MockPubSub{}}

	from := "2"
	to := "1"
	n := 50
	amount := int64(10)

	var fromBalanceBefore, toBalanceBefore int64
	require.NoError(t, db.Table("users").Select("balance").Where("id = ?", from).Scan(&fromBalanceBefore).Error)
	require.NoError(t, db.Table("users").Select("balance").Where("id = ?", to).Scan(&toBalanceBefore).Error)

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

	var fromBalanceAfter, toBalanceAfter int64
	require.NoError(t, db.Table("users").Select("balance").Where("id = ?", from).Scan(&fromBalanceAfter).Error)
	require.NoError(t, db.Table("users").Select("balance").Where("id = ?", to).Scan(&toBalanceAfter).Error)

	expectedFrom := fromBalanceBefore - int64(n)*amount
	expectedTo := toBalanceBefore + int64(n)*amount

	require.Equal(t, expectedFrom, fromBalanceAfter, "Số dư người gửi không đúng")
	require.Equal(t, expectedTo, toBalanceAfter, "Số dư người nhận không đúng")

	totalBefore := fromBalanceBefore + toBalanceBefore
	totalAfter := fromBalanceAfter + toBalanceAfter
	require.Equal(t, totalBefore, totalAfter, "Tổng tiền trong hệ thống phải không đổi")
}

func TestInsertTransaction_InvalidUsers(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormTransferRepo{db: db, pubsub: &MockPubSub{}}

	ctx := context.Background()

	err := repo.InsertTransaction(ctx, "nonexistent", "1", 100)
	require.Error(t, err)

	err = repo.InsertTransaction(ctx, "1", "nonexistent", 100)
	require.Error(t, err)
}
