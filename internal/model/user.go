package model

//go:generate goqueryset -in user.go

// gen:qs
type User struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	Name     string `gorm:"not null"`
	Balance  int64  `gorm:"not null;default:0"`
	Password string `gorm:"not null"`
}
