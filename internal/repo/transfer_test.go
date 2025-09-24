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

func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "host=localhost user=demo_user password=demo_pass dbname=demo_db sslmode=disable connect_timeout=10"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "failed to open GORM v2 DB")

	sqlDB, err := db.DB()
	require.NoError(t, err, "failed to get sql.DB from GORM")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, sqlDB.PingContext(ctx), "failed to connect to test database")

	return db
}
func TestInsertTransaction_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormTransferRepo{db: db}

	ctx := context.Background()
	from := int64(1)
	to := int64(2)
	amount := int64(2)

	fromBalanceBefore, err := repo.GetBalance(ctx, from)
	require.NoError(t, err)
	toBalanceBefore, err := repo.GetBalance(ctx, to)
	require.NoError(t, err)
	fmt.Printf("[Before] From: %d, To: %d\n", fromBalanceBefore, toBalanceBefore)

	err = repo.InsertTransaction(ctx, from, to, amount)
	require.NoError(t, err)

	fromBalanceAfter, err := repo.GetBalance(ctx, from)
	require.NoError(t, err)
	toBalanceAfter, err := repo.GetBalance(ctx, to)
	require.NoError(t, err)
	fmt.Printf("[After]  From: %d, To: %d\n", fromBalanceAfter, toBalanceAfter)

	require.Equal(t, fromBalanceBefore-amount, fromBalanceAfter)
	require.Equal(t, toBalanceBefore+amount, toBalanceAfter)
}

func TestInsertTransaction_InsufficientBalance(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormTransferRepo{db: db}

	ctx := context.Background()
	from := int64(1)
	to := int64(2)
	amount := int64(2000000)

	fromBalanceBefore, err := repo.GetBalance(ctx, from)
	require.NoError(t, err)
	toBalanceBefore, err := repo.GetBalance(ctx, to)
	require.NoError(t, err)

	var countBefore int64
	require.NoError(t, db.Table("transactions").Count(&countBefore).Error)
	fmt.Printf("[Before] From: %d, To: %d\n", fromBalanceBefore, toBalanceBefore)

	err = repo.InsertTransaction(ctx, from, to, amount)
	require.Error(t, err)

	fromBalanceAfter, err := repo.GetBalance(ctx, from)
	require.NoError(t, err)
	toBalanceAfter, err := repo.GetBalance(ctx, to)
	require.NoError(t, err)

	require.Equal(t, fromBalanceBefore, fromBalanceAfter)
	require.Equal(t, toBalanceBefore, toBalanceAfter)
	fmt.Printf("[After] From: %d, To: %d\n", fromBalanceAfter, toBalanceAfter)

	var countAfter int64
	require.NoError(t, db.Table("transactions").Count(&countAfter).Error)
	require.Equal(t, countBefore, countAfter)
}

func TestInsertTransaction_Concurrent(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormTransferRepo{db: db}

	from := int64(2)
	to := int64(1)
	n := 50
	amount := int64(1)

	ctx := context.Background()
	fromBalanceBefore, err := repo.GetBalance(ctx, from)
	require.NoError(t, err)
	toBalanceBefore, err := repo.GetBalance(ctx, to)
	require.NoError(t, err)
	fmt.Printf("[Before Concurrent] From: %d, To: %d\n", fromBalanceBefore, toBalanceBefore)

	var wg sync.WaitGroup
	errs := make(chan error, n)
	startLine := make(chan struct{})

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(job int) {
			defer wg.Done()
			<-startLine

			jobCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
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
	fmt.Printf("[After Concurrent] From: %d, To: %d\n", fromBalanceAfter, toBalanceAfter)

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
	repo := &GormTransferRepo{db: db}

	ctx := context.Background()

	err := repo.InsertTransaction(ctx, -1, 1, 100)
	require.Error(t, err)

	err = repo.InsertTransaction(ctx, 1, -1, 100)
	require.Error(t, err)
}

func TestInsertTransaction_ConcurrentDeadlock(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := &GormTransferRepo{db: db}

	userA := int64(1)
	userB := int64(2)
	amount := int64(1)
	amount2 := int64(10)

	balanceA1, err := repo.GetBalance(ctx, userA)
	require.NoError(t, err)
	balanceB1, err := repo.GetBalance(ctx, userB)
	require.NoError(t, err)
	fmt.Println(balanceA1, " ", balanceB1)
	var wg sync.WaitGroup
	errs := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := repo.InsertTransaction(ctx, userA, userB, amount); err != nil {
			errs <- fmt.Errorf("A→B failed: %w", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := repo.InsertTransaction(ctx, userB, userA, amount); err != nil {
			errs <- fmt.Errorf("B→A failed: %w", err)
		}
	}()

	wg.Wait()
	close(errs)

	var errorCount int
	for err := range errs {
		t.Log(err)
		errorCount++
	}
	balanceA2, err := repo.GetBalance(ctx, userA)
	require.NoError(t, err)
	balanceB2, err := repo.GetBalance(ctx, userB)
	require.NoError(t, err)
	fmt.Println(balanceA2, " ", balanceB2)

	require.Equal(t, balanceA1-amount+amount2, balanceA2, "UserA balance mismatch")
	require.Equal(t, balanceB1+amount-amount2, balanceB2, "UserB balance mismatch")
	require.Equal(t, 0, errorCount, "phát hiện lỗi trong giao dịch đồng thời")

}
