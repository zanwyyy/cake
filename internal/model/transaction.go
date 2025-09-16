package model

type Transaction struct {
	ID     uint   `gorm:"primaryKey;autoIncrement"`
	From   string `gorm:"column:from_user;not null"`
	To     string `gorm:"column:to_user;not null"`
	Amount int64  `gorm:"not null"`
}
