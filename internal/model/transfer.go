package model

type ListTransactionsInput struct {
	UserId int64
}

type ListTransactionsOutput struct {
	Number       int64
	Transactions []Transaction
}

type SendMoneyInput struct {
	From   int64
	To     int64
	Amount int64
}

type SendMoneyOutput struct {
	Success      bool
	ErrorMessage string
}

type GetBalanceInput struct {
	UserId int64
}

type GetBalanceOutput struct {
	UserId  int64
	Balance int64
}
