package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/antonkazachenko/go-todo-list-api/internal/entities"
	"github.com/antonkazachenko/go-todo-list-api/internal/storage/sqlite"
	"github.com/antonkazachenko/go-todo-list-api/models"
	"github.com/antonkazachenko/go-todo-list-api/utils"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const format = "20060102"

type Handlers struct {
	TaskRepo storage.SQLiteTaskRepository
}

func (h *Handlers) NextDate(now time.Time, date string, repeat string) (string, error) {
	parsedDate, err := time.Parse(format, date)
	if err != nil {
		return "", errors.New("недопустимый формат date")
	}

	repeatParts := strings.SplitN(repeat, " ", 2)
	repeatType := ""
	repeatRule := ""

	if len(repeatParts) > 0 {
		repeatType = repeatParts[0]
	}
	if len(repeatParts) > 1 {
		repeatRule = repeatParts[1]
	}

	if repeatType == "d" {
		if repeatRule == "" {
			return "", errors.New("не указан интервал в днях")
		} else {
			numberOfDays, err := strconv.Atoi(repeatRule)
			if err != nil {
				return "", errors.New("некорректно указано правило repeat")
			}
			if numberOfDays > 400 {
				return "", errors.New("превышен максимально допустимый интервал")
			}

			if now.Format(format) != parsedDate.Format(format) {
				if now.After(parsedDate) {
					for now.After(parsedDate) || now.Format(format) == parsedDate.Format(format) {
						parsedDate = parsedDate.AddDate(0, 0, numberOfDays)
					}
				} else {
					parsedDate = parsedDate.AddDate(0, 0, numberOfDays)
				}
			}

		}
	} else if repeatType == "y" {
		parsedDate = parsedDate.AddDate(1, 0, 0)
		for now.After(parsedDate) {
			parsedDate = parsedDate.AddDate(1, 0, 0)
		}
	} else if repeatType == "w" {
		substrings := strings.Split(repeatRule, ",")

		daysOfWeek := make(map[int]bool)
		for _, value := range substrings {
			number, err := strconv.Atoi(value)
			if err != nil {
				return "", errors.New("ошибка конвертации значения дня недели")
			}
			if number < 1 || number > 7 {
				return "", errors.New("недопустимое значение дня недели")
			}

			if number == 7 {
				number = 0
			}
			daysOfWeek[number] = true
		}

		for {
			if daysOfWeek[int(parsedDate.Weekday())] {
				if now.Before(parsedDate) {
					break
				}
			}
			parsedDate = parsedDate.AddDate(0, 0, 1)
		}
	} else if repeatType == "m" {
		repeatParts := strings.Split(repeatRule, " ")
		daysPart := repeatParts[0]
		monthsPart := ""
		if len(repeatParts) > 1 {
			monthsPart = repeatParts[1]
		}

		days := strings.Split(daysPart, ",")
		months := strings.Split(monthsPart, ",")

		dayMap := make(map[int]bool)
		for _, dayStr := range days {
			day, err := strconv.Atoi(dayStr)
			if err != nil {
				return "", errors.New("ошибка конвертации значения дня месяца")
			}
			if day < -2 || day > 31 || day == 0 {
				return "", errors.New("недопустимое значение дня месяца")
			}
			dayMap[day] = true
		}

		monthMap := make(map[int]bool)
		for _, monthStr := range months {
			if monthStr != "" {
				month, err := strconv.Atoi(monthStr)
				if err != nil {
					return "", errors.New("ошибка конвертации значения месяца")
				}
				if month < 1 || month > 12 {
					return "", errors.New("недопустимое значение месяца")
				}
				monthMap[month] = true
			}
		}

		found := false
		for i := 0; i < 12*10; i++ {
			month := int(parsedDate.Month())
			if len(monthMap) > 0 && !monthMap[month] {
				parsedDate = parsedDate.AddDate(0, 1, 0)
				parsedDate = time.Date(parsedDate.Year(), parsedDate.Month(), 1, 0, 0, 0, 0, parsedDate.Location())
				continue
			}

			lastDayOfMonth := time.Date(parsedDate.Year(), parsedDate.Month()+1, 0, 0, 0, 0, 0, parsedDate.Location()).Day()
			for targetDay := range dayMap {
				if targetDay > 0 {
					if parsedDate.Day() == targetDay && now.Before(parsedDate) {
						found = true
						break
					}
				} else if targetDay < 0 {
					if parsedDate.Day() == lastDayOfMonth+targetDay+1 && now.Before(parsedDate) {
						found = true
						break
					}
				}
			}
			if found {
				break
			}

			parsedDate = parsedDate.AddDate(0, 0, 1)
			if parsedDate.Day() == 1 {
				parsedDate = time.Date(parsedDate.Year(), parsedDate.Month(), 1, 0, 0, 0, 0, parsedDate.Location())
			}
		}

		if !found {
			return "", nil
		}
	} else {
		return "", errors.New("недопустимый символ")
	}

	return parsedDate.Format(format), nil
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
		dateInTime, err = time.Parse(format, task.Date)
		if err != nil {
			utils.SendErrorResponse(res, "недопустимый формат date", http.StatusBadRequest)
			return
		}
	} else {
		task.Date = time.Now().Format(format)
		dateInTime = time.Now()
	}

	if time.Now().After(dateInTime) {
		if task.Repeat == "" {
			task.Date = time.Now().Format(format)
			dateInTime = time.Now()
		} else {
			task.Date, err = h.NextDate(time.Now(), task.Date, task.Repeat)
			if err != nil {
				utils.SendErrorResponse(res, err.Error(), http.StatusBadRequest)
				return
			}
		}
	}

	id, err := h.TaskRepo.AddTask(task)
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

	tasks, err := h.TaskRepo.GetTasks(searchTerm, limit)
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

	task, err := h.TaskRepo.GetTaskByID(id)
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

	_, ok := taskUpdates["id"].(string)
	if !ok {
		utils.SendErrorResponse(res, "отсутствует обязательное поле id", http.StatusBadRequest)
		return
	}

	_, err = h.TaskRepo.UpdateTask(taskUpdates)
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

	_, err := h.TaskRepo.DeleteTask(id)
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

	task, err := h.TaskRepo.GetTaskByID(id)
	if err != nil {
		utils.SendErrorResponse(res, "задача с указанным id не найдена", http.StatusNotFound)
		return
	}

	if task.Repeat == "" {
		_, err = h.TaskRepo.DeleteTask(id)
		if err != nil {
			utils.SendErrorResponse(res, "ошибка запроса к базе данных", http.StatusInternalServerError)
			return
		}
	} else {
		parsedDate, err := time.Parse(format, task.Date)
		if err != nil {
			utils.SendErrorResponse(res, "недопустимый формат date", http.StatusBadRequest)
			return
		}

		if parsedDate.Format(format) == time.Now().Format(format) {
			parsedDate = parsedDate.AddDate(0, 0, -1)
			task.Date, err = h.NextDate(parsedDate, task.Date, task.Repeat)
		} else {
			task.Date, err = h.NextDate(time.Now(), task.Date, task.Repeat)
		}

		if err != nil {
			utils.SendErrorResponse(res, err.Error(), http.StatusBadRequest)
			return
		}

		err = h.TaskRepo.MarkTaskAsDone(id, task.Date)
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
