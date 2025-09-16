package model

type TransactionEvent struct {
	From   string
	To     string
	Amount int64
	Status string
}
