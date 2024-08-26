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
	var task entities.Task
	if err := parseRequestBody(req, &task); err != nil {
		utils.SendErrorResponse(res, "ошибка декодирования JSON", http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		utils.SendErrorResponse(res, "отсутствует обязательное поле title", http.StatusBadRequest)
		return
	}

	if err := h.validateAndUpdateDate(&task); err != nil {
		utils.SendErrorResponse(res, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.TaskService.Repo.AddTask(task)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка запроса к базе данных", http.StatusInternalServerError)
		return
	}

	sendJSONResponse(res, http.StatusOK, models.IDResponse{ID: id})
}

func (h *Handlers) HandleGetTasks(res http.ResponseWriter, req *http.Request) {
	searchTerm := req.URL.Query().Get("search")
	limit := 100

	tasks, err := h.TaskService.Repo.GetTasks(searchTerm, limit)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка запроса к базе данных", http.StatusInternalServerError)
		return
	}

	if tasks == nil || len(tasks) == 0 {
		tasks = []entities.Task{}
	}

	sendJSONResponse(res, http.StatusOK, map[string][]entities.Task{"tasks": tasks})
}

func (h *Handlers) HandleGetTask(res http.ResponseWriter, req *http.Request) {
	taskID, err := parseAndValidateID(req.URL.Query().Get("id"))
	if err != nil {
		utils.SendErrorResponse(res, err.Error(), http.StatusBadRequest)
		return
	}

	task, err := h.TaskService.Repo.GetTaskByID(taskID)
	if err != nil {
		utils.SendErrorResponse(res, "задача с указанным id не найдена", http.StatusNotFound)
		return
	}

	sendJSONResponse(res, http.StatusOK, task)
}

func (h *Handlers) HandlePutTask(res http.ResponseWriter, req *http.Request) {
	var taskUpdates map[string]interface{}
	if err := parseRequestBody(req, &taskUpdates); err != nil {
		utils.SendErrorResponse(res, "ошибка декодирования JSON", http.StatusBadRequest)
		return
	}

	_, err := h.validateAndExtractID(taskUpdates)
	if err != nil {
		utils.SendErrorResponse(res, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.validateTaskUpdates(taskUpdates); err != nil {
		utils.SendErrorResponse(res, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := h.TaskService.Repo.UpdateTask(taskUpdates); err != nil {
		utils.SendErrorResponse(res, "ошибка запроса к базе данных", http.StatusInternalServerError)
		return
	}

	sendJSONResponse(res, http.StatusOK, map[string]interface{}{})
}

func (h *Handlers) HandleDeleteTask(res http.ResponseWriter, req *http.Request) {
	taskID, err := parseAndValidateID(req.URL.Query().Get("id"))
	if err != nil {
		utils.SendErrorResponse(res, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.deleteTaskIfExists(taskID); err != nil {
		utils.SendErrorResponse(res, err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(res, http.StatusOK, map[string]interface{}{})
}

func (h *Handlers) HandleDoneTask(res http.ResponseWriter, req *http.Request) {
	taskID, err := parseAndValidateID(req.URL.Query().Get("id"))
	if err != nil {
		utils.SendErrorResponse(res, err.Error(), http.StatusBadRequest)
		return
	}

	task, err := h.TaskService.Repo.GetTaskByID(taskID)
	if err != nil {
		utils.SendErrorResponse(res, "задача с указанным id не найдена", http.StatusNotFound)
		return
	}

	if task.Repeat == "" {
		if _, err := h.TaskService.Repo.DeleteTask(taskID); err != nil {
			utils.SendErrorResponse(res, "ошибка запроса к базе данных", http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.markTaskAsDone(taskID, task); err != nil {
			utils.SendErrorResponse(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	sendJSONResponse(res, http.StatusOK, map[string]interface{}{})
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

	newDate, err := h.TaskService.NextDate(now, date, repeat)
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

func parseRequestBody(req *http.Request, target interface{}) error {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(req.Body); err != nil {
		return fmt.Errorf("ошибка чтения тела запроса")
	}
	return json.Unmarshal(buf.Bytes(), target)
}

func sendJSONResponse(res http.ResponseWriter, statusCode int, data interface{}) {
	respBytes, err := json.Marshal(data)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(statusCode)
	_, err = res.Write(respBytes)
	if err != nil {
		utils.SendErrorResponse(res, "ошибка записи ответа", http.StatusInternalServerError)
		return
	}
}

func parseAndValidateID(idStr string) (string, error) {
	if idStr == "" {
		return "", fmt.Errorf("не передан идентификатор")
	}

	if _, err := strconv.ParseInt(idStr, 10, 64); err != nil {
		return "", fmt.Errorf("id должен быть числом")
	}
	return idStr, nil
}

func (h *Handlers) validateAndExtractID(taskUpdates map[string]interface{}) (string, error) {
	id, ok := taskUpdates["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("отсутствует обязательное поле id")
	}

	if _, err := strconv.ParseInt(id, 10, 64); err != nil {
		return "", fmt.Errorf("id должен быть числом")
	}

	if _, err := h.TaskService.Repo.GetTaskByID(id); err != nil {
		return "", fmt.Errorf("задача с указанным id не найдена")
	}

	return id, nil
}

func (h *Handlers) validateTaskUpdates(taskUpdates map[string]interface{}) error {
	date, dateOk := taskUpdates["date"].(string)
	if title, ok := taskUpdates["title"].(string); !ok || strings.TrimSpace(title) == "" {
		return fmt.Errorf("отсутствует обязательное поле title")
	}

	if !dateOk || strings.TrimSpace(date) == "" {
		return fmt.Errorf("отсутствует обязательное поле date")
	}

	if _, err := time.Parse(service.Format, date); err != nil {
		return fmt.Errorf("недопустимый формат date")
	}

	if repeat, ok := taskUpdates["repeat"].(string); ok && strings.TrimSpace(repeat) != "" {
		repeatParts := strings.SplitN(repeat, " ", 2)
		repeatType := repeatParts[0]
		if !isValidRepeatType(repeatType) {
			return fmt.Errorf("недопустимый символ")
		}
	}

	return nil
}

func (h *Handlers) validateAndUpdateDate(task *entities.Task) error {
	var dateInTime time.Time
	var err error

	if task.Date != "" {
		dateInTime, err = time.Parse(service.Format, task.Date)
		if err != nil {
			return fmt.Errorf("недопустимый формат date")
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
				return err
			}
		}
	}

	return nil
}

func (h *Handlers) markTaskAsDone(taskID string, task *entities.Task) error {
	parsedDate, err := time.Parse(service.Format, task.Date)
	if err != nil {
		return fmt.Errorf("недопустимый формат date")
	}

	if parsedDate.Format(service.Format) == time.Now().Format(service.Format) {
		parsedDate = parsedDate.AddDate(0, 0, -1)
		task.Date, err = h.TaskService.NextDate(parsedDate, task.Date, task.Repeat)
	} else {
		task.Date, err = h.TaskService.NextDate(time.Now(), task.Date, task.Repeat)
	}

	if err != nil {
		return err
	}

	if err := h.TaskService.Repo.MarkTaskAsDone(taskID, task.Date); err != nil {
		return fmt.Errorf("ошибка при обновлении задачи")
	}

	return nil
}

func (h *Handlers) deleteTaskIfExists(taskID string) error {
	if _, err := h.TaskService.Repo.GetTaskByID(taskID); err != nil {
		return fmt.Errorf("задача с указанным id не найдена")
	}

	if _, err := h.TaskService.Repo.DeleteTask(taskID); err != nil {
		return fmt.Errorf("ошибка запроса к базе данных")
	}

	return nil
}

func isValidRepeatType(repeatType string) bool {
	validTypes := []string{"d", "w", "m", "y"}
	for _, v := range validTypes {
		if repeatType == v {
			return true
		}
	}
	return false
}
