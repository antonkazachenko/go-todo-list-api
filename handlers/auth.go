package handlers

import (
	"bytes"
	"encoding/json"
	"github.com/antonkazachenko/go-todo-list-api/models"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"os"
)

func (h *Handlers) HandleSignIn(res http.ResponseWriter, req *http.Request) {
	pass := os.Getenv("TODO_PASSWORD")
	if len(pass) == 0 {
		var resp models.ErrorResponse
		resp.Error = "пароль не установлен"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		var resp models.ErrorResponse
		resp.Error = "ошибка чтения тела запроса"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
			return
		}
		http.Error(res, string(respBytes), http.StatusBadRequest)
		return
	}

	var body map[string]string
	err = json.Unmarshal(buf.Bytes(), &body)
	if err != nil {
		var resp models.ErrorResponse
		resp.Error = "ошибка десериализации JSON"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
			return
		}
		http.Error(res, string(respBytes), http.StatusBadRequest)
		return
	}

	if body["password"] != pass {
		var resp models.ErrorResponse
		resp.Error = "неверный пароль"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
			return
		}
		http.Error(res, string(respBytes), http.StatusBadRequest)
		return
	}

	token := jwt.New(jwt.SigningMethodHS256)
	tokenString, err := token.SignedString([]byte(pass))
	if err != nil {
		var resp models.ErrorResponse
		resp.Error = "ошибка создания токена"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
		return
	}

	var resp models.AuthResponse
	resp.Token = tokenString
	respBytes, err := json.Marshal(resp)
	if err != nil {
		http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(respBytes)
	if err != nil {
		http.Error(res, "ошибка записи ответа", http.StatusInternalServerError)
	}
}
