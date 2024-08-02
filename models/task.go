package models

type Task struct {
	ID      string `json:"id"`
	Date    string `json:"date,omitempty"`
	Title   string `json:"title"`
	Comment string `json:"comment,omitempty"`
	Repeat  string `json:"repeat,omitempty"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

type IDResponse struct {
	ID int64 `json:"id"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
