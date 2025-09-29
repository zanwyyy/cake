package model

type LoginInput struct {
	Username int64
	Password string
}

type LoginOutput struct {
	AccessToken  string
	RefreshToken string
}

type RefreshInput struct {
	RefreshToken string
}

type RefreshOutput struct {
	AccessToken string
}
