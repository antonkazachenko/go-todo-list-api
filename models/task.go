package models

type AuthResponse struct {
	Token string `json:"token"`
}

type IDResponse struct {
	ID int64 `json:"id"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
