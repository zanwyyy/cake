package model

type LoginInput struct {
	Username int64
	Password string
}

type LoginOutput struct {
	AccessToken string
}

type LogoutInput struct {
	UserID int64
}

type LogoutOutput struct {
	Success bool
}
