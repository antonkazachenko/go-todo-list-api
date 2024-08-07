package utils

import (
	"encoding/json"
	"net/http"

	"github.com/antonkazachenko/go-todo-list-api/models"
)

func SendErrorResponse(res http.ResponseWriter, errorMessage string, statusCode int) {
	resp := models.ErrorResponse{Error: errorMessage}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
		return
	}
	http.Error(res, string(respBytes), statusCode)
}
