package utils

import "fmt"

func ValidateUserID(userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user_id: must be greater than 0")
	}
	return nil
}

func ValidateAmount(amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("invalid amount: must be greater than 0")
	}
	if amount >= 1_000_000_000 {
		return fmt.Errorf("invalid amount: must be less than 1_000_000_000")
	}
	return nil
}
