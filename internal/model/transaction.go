package model

//go:generate goqueryset -in transaction.go

// gen:qs
type Transaction struct {
	ID     uint  `gorm:"primaryKey;autoIncrement"`
	From   int64 `gorm:"column:from_user;not null"` // ID của user gửi
	To     int64 `gorm:"column:to_user;not null"`   // ID của user nhận
	Amount int64 `gorm:"not null"`
}
