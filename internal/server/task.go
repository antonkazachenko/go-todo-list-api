package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/antonkazachenko/go-todo-list-api/internal/entities"
	"github.com/antonkazachenko/go-todo-list-api/internal/service"
	"github.com/antonkazachenko/go-todo-list-api/models"
	"github.com/antonkazachenko/go-todo-list-api/utils"
)

type Handlers struct {
	TaskService *service.TaskService
}

func NewHandlers(taskService *service.TaskService) *Handlers {
	return &Handlers{TaskService: taskService}
}

func (h *Handlers) HandleAddTask(res http.ResponseWriter, req *http.Request) {
	var buf bytes.Buffer
	var task entities.Task

	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка чтения тела запроса", http.StatusBadRequest)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &task); err != nil {
		utils.SendErrorResponse(res, "ошибка декодирования JSON", http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		utils.SendErrorResponse(res, "отсутствует обязательное поле title", http.StatusBadRequest)
		return
	}

	var dateInTime time.Time
	if task.Date != "" {
		dateInTime, err = time.Parse(service.Format, task.Date)
		if err != nil {
			utils.SendErrorResponse(res, "недопустимый формат date", http.StatusBadRequest)
			return
		}
	} else {
		task.Date = time.Now().Format(service.Format)
		dateInTime = time.Now()
	}

	if time.Now().After(dateInTime) {
		if task.Repeat == "" {
			task.Date = time.Now().Format(service.Format)
			dateInTime = time.Now()
		} else {
			task.Date, err = h.TaskService.NextDate(time.Now(), task.Date, task.Repeat)
			if err != nil {
				utils.SendErrorResponse(res, err.Error(), http.StatusBadRequest)
				return
			}
		}
	}

	id, err := h.TaskService.Repo.AddTask(task)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка запроса к базе данных", http.StatusInternalServerError)
		return
	}

	resp := models.IDResponse{ID: id}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		http.Error(res, fmt.Sprintf("ошибка при сериализации ответа: %v", err), http.StatusBadRequest)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(respBytes)
	if err != nil {
		http.Error(res, fmt.Sprintf("ошибка при записи ответа: %v", err), http.StatusBadRequest)
		return
	}
}

func (h *Handlers) HandleGetTasks(res http.ResponseWriter, req *http.Request) {
	searchTerm := req.URL.Query().Get("search")
	limit := 100

	tasks, err := h.TaskService.Repo.GetTasks(searchTerm, limit)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка запроса к базе данных", http.StatusInternalServerError)
		return
	}

	if tasks == nil {
		tasks = []entities.Task{}
	}
	respBytes, err := json.Marshal(map[string][]entities.Task{"tasks": tasks})
	if err != nil {
		utils.SendErrorResponse(res, "ошибка сериализации ответа", http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(respBytes)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка записи ответа", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) HandleGetTask(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")

	if id == "" {
		utils.SendErrorResponse(res, "не передан идентификатор", http.StatusBadRequest)
		return
	}

	task, err := h.TaskService.Repo.GetTaskByID(id)
	if err != nil {
		utils.SendErrorResponse(res, "задача с указанным id не найдена", http.StatusNotFound)
		return
	}

	respBytes, err := json.Marshal(task)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка сериализации ответа", http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(respBytes)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка записи ответа", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) HandlePutTask(res http.ResponseWriter, req *http.Request) {
	var taskUpdates map[string]interface{}

	err := json.NewDecoder(req.Body).Decode(&taskUpdates)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка декодирования JSON", http.StatusBadRequest)
		return
	}

	id, ok := taskUpdates["id"].(string)
	if !ok || id == "" {
		utils.SendErrorResponse(res, "отсутствует обязательное поле id", http.StatusBadRequest)
		return
	}

	_, err = strconv.ParseInt(id, 10, 64)
	if err != nil {
		utils.SendErrorResponse(res, "id должен быть числом", http.StatusBadRequest)
		return
	}

	_, err = h.TaskService.Repo.GetTaskByID(id)
	if err != nil {
		utils.SendErrorResponse(res, "задача с указанным id не найдена", http.StatusNotFound)
		return
	}

	title, ok := taskUpdates["title"].(string)
	if !ok || title == "" {
		utils.SendErrorResponse(res, "отсутствует обязательное поле title", http.StatusBadRequest)
		return
	}

	date, ok := taskUpdates["date"].(string)
	if !ok || date == "" {
		utils.SendErrorResponse(res, "отсутствует обязательное поле date", http.StatusBadRequest)
		return
	}

	_, err = time.Parse("20060102", date)
	if err != nil {
		utils.SendErrorResponse(res, "недопустимый формат date", http.StatusBadRequest)
		return
	}

	if repeat, ok := taskUpdates["repeat"].(string); ok {
		if repeat != "" {
			repeatParts := strings.SplitN(repeat, " ", 2)
			repeatType := repeatParts[0]
			if repeatType != "d" && repeatType != "w" && repeatType != "m" && repeatType != "y" {
				utils.SendErrorResponse(res, "недопустимый символ", http.StatusBadRequest)
				return
			}
		}
	}

	_, err = h.TaskService.Repo.UpdateTask(taskUpdates)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка запроса к базе данных", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write([]byte(`{}`))
	if err != nil {
		utils.SendErrorResponse(res, "ошибка записи ответа", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) HandleDeleteTask(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")

	if id == "" {
		utils.SendErrorResponse(res, "не передан идентификатор", http.StatusBadRequest)
		return
	}

	_, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		utils.SendErrorResponse(res, "id должен быть числом", http.StatusBadRequest)
		return
	}

	_, err = h.TaskService.Repo.GetTaskByID(id)
	if err != nil {
		utils.SendErrorResponse(res, "задача с указанным id не найдена", http.StatusNotFound)
		return
	}

	_, err = h.TaskService.Repo.DeleteTask(id)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка запроса к базе данных", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	respBytes, err := json.Marshal(struct{}{})
	if err != nil {
		http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
		return
	}
	_, err = res.Write(respBytes)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка записи ответа", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) HandleDoneTask(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")

	if id == "" {
		utils.SendErrorResponse(res, "не передан идентификатор", http.StatusBadRequest)
		return
	}

	task, err := h.TaskService.Repo.GetTaskByID(id)
	if err != nil {
		utils.SendErrorResponse(res, "задача с указанным id не найдена", http.StatusNotFound)
		return
	}

	if task.Repeat == "" {
		_, err = h.TaskService.Repo.DeleteTask(id)
		if err != nil {
			utils.SendErrorResponse(res, "ошибка запроса к базе данных", http.StatusInternalServerError)
			return
		}
	} else {
		parsedDate, err := time.Parse(service.Format, task.Date)
		if err != nil {
			utils.SendErrorResponse(res, "недопустимый формат date", http.StatusBadRequest)
			return
		}

		if parsedDate.Format(service.Format) == time.Now().Format(service.Format) {
			parsedDate = parsedDate.AddDate(0, 0, -1)
			task.Date, err = h.TaskService.NextDate(parsedDate, task.Date, task.Repeat)
		} else {
			task.Date, err = h.TaskService.NextDate(time.Now(), task.Date, task.Repeat)
		}

		if err != nil {
			utils.SendErrorResponse(res, err.Error(), http.StatusBadRequest)
			return
		}

		err = h.TaskService.Repo.MarkTaskAsDone(id, task.Date)
		if err != nil {
			utils.SendErrorResponse(res, "ошибка запроса к базе данных", http.StatusInternalServerError)
			return
		}
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	respBytes, err := json.Marshal(struct{}{})
	if err != nil {
		http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
		return
	}
	_, err = res.Write(respBytes)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка записи ответа", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) HandleNextDate(res http.ResponseWriter, req *http.Request) {
	nowParam := req.URL.Query().Get("now")
	date := req.URL.Query().Get("date")
	repeat := req.URL.Query().Get("repeat")

	now, err := time.Parse(service.Format, nowParam)

	if err != nil {
		http.Error(res, "Неправильный формат парамeтра now", http.StatusBadRequest)
		return
	}
	var newDate string
	if repeat != "" {
		newDate, err = h.TaskService.NextDate(now, date, repeat)
	}
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write([]byte(newDate))
	if err != nil {
		fmt.Printf("Error in writing a response for /api/nextdate GET request,\n %v", err)
		return
	}
}
