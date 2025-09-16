package model

type User struct {
	ID      uint   `gorm:"primaryKey;autoIncrement"`
	Name    string `gorm:"not null"`
	Balance int64  `gorm:"not null;default:0"`
}
