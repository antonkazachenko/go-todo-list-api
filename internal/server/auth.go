package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/antonkazachenko/go-todo-list-api/config"
	"github.com/antonkazachenko/go-todo-list-api/models"
	"github.com/antonkazachenko/go-todo-list-api/utils"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
)

func (h *Handlers) HandleSignIn(res http.ResponseWriter, req *http.Request) {
	pass := config.TODO_PASS
	if len(pass) == 0 {
		utils.SendErrorResponse(res, "пароль не установлен", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка чтения тела запроса", http.StatusBadRequest)
		return
	}

	var body map[string]string
	err = json.Unmarshal(buf.Bytes(), &body)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка декодирования JSON", http.StatusBadRequest)
		return
	}

	if body["password"] != pass {
		utils.SendErrorResponse(res, "неверный пароль", http.StatusUnauthorized)
		return
	}

	token := jwt.New(jwt.SigningMethodHS256)
	tokenString, err := token.SignedString([]byte(pass))
	if err != nil {
		utils.SendErrorResponse(res, "ошибка создания токена", http.StatusInternalServerError)
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
		fmt.Println("ошибка записи ответа в HandleSignIn", err)
	}
}
