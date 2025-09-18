package model

type Transaction struct {
	ID     uint  `gorm:"primaryKey;autoIncrement"`
	From   int64 `gorm:"column:from_user;not null"`
	To     int64 `gorm:"column:to_user;not null"`
	Amount int64 `gorm:"not null"`
}
