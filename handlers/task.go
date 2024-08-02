package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/antonkazachenko/go-todo-list-api/models"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Handlers struct {
	DB *sql.DB
}

func (h *Handlers) NextDate(now time.Time, date string, repeat string) (string, error) {
	parsedDate, err := time.Parse("20060102", date)
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

			if now.Format("20060102") != parsedDate.Format("20060102") {
				if now.After(parsedDate) {
					for now.After(parsedDate) || now.Format("20060102") == parsedDate.Format("20060102") {
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

	return parsedDate.Format("20060102"), nil
}

func (h *Handlers) HandleAddTask(res http.ResponseWriter, req *http.Request) {
	var buf bytes.Buffer
	var task models.Task

	_, err := buf.ReadFrom(req.Body)
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

	if err = json.Unmarshal(buf.Bytes(), &task); err != nil {
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

	if task.Title == "" {
		var resp models.ErrorResponse
		resp.Error = "отсутствует обязательное поле title"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
			return
		}
		http.Error(res, string(respBytes), http.StatusBadRequest)
		return
	}

	var dateInTime time.Time
	if task.Date != "" {
		dateInTime, err = time.Parse("20060102", task.Date)
		if err != nil {
			var resp models.ErrorResponse
			resp.Error = "недопустимый формат date"
			respBytes, err := json.Marshal(resp)
			if err != nil {
				http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
				return
			}
			http.Error(res, string(respBytes), http.StatusBadRequest)
			return
		}
	} else {
		task.Date = time.Now().Format("20060102")
		dateInTime = time.Now()
	}

	if time.Now().After(dateInTime) {
		if task.Repeat == "" {
			task.Date = time.Now().Format("20060102")
			dateInTime = time.Now()
		} else {
			task.Date, err = h.NextDate(time.Now(), task.Date, task.Repeat)
			if err != nil {
				var resp models.ErrorResponse
				resp.Error = "правило повторения указано в неправильном формате"
				respBytes, err := json.Marshal(resp)
				if err != nil {
					http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
					return
				}
				http.Error(res, string(respBytes), http.StatusBadRequest)
				return
			}
		}
	}

	result, err := h.DB.Exec("INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)",
		task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		var resp models.ErrorResponse
		resp.Error = "ошибка запроса к базе данных"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
			return
		}
		http.Error(res, string(respBytes), http.StatusBadRequest)
		return
	}

	var ret models.IDResponse
	ret.ID, err = result.LastInsertId()
	if err != nil {
		http.Error(res, fmt.Sprintf("ошибка получения последнего добавленного ID: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := json.Marshal(ret)
	if err != nil {
		http.Error(res, fmt.Sprintf("ошибка при сериализации ответа: %v", err), http.StatusBadRequest)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(resp)
	if err != nil {
		http.Error(res, fmt.Sprintf("ошибка при записи ответа: %v", err), http.StatusBadRequest)
		return
	}
}

func (h *Handlers) HandleGetTasks(res http.ResponseWriter, req *http.Request) {
	searchTerm := req.URL.Query().Get("search")
	limit := 100

	query := "SELECT id, date, title, comment, repeat FROM scheduler"
	args := []interface{}{}

	parsedDate, dateErr := time.Parse("02.01.2006", searchTerm)
	if dateErr == nil {
		formattedDate := parsedDate.Format("20060102")
		query += " WHERE date = ? ORDER BY date LIMIT ?"
		args = append(args, formattedDate, limit)
	} else if searchTerm != "" {
		query += " WHERE title LIKE ? OR comment LIKE ? ORDER BY date LIMIT ?"
		searchTerm = "%" + searchTerm + "%"
		args = append(args, searchTerm, searchTerm, limit)
	} else {
		query += " ORDER BY date LIMIT ?"
		args = append(args, limit)
	}
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		var resp models.ErrorResponse
		resp.Error = "ошибка запроса к базе данных"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []map[string]string
	for rows.Next() {
		var id int64
		var date, title, comment, repeat string
		err = rows.Scan(&id, &date, &title, &comment, &repeat)
		if err != nil {
			var resp models.ErrorResponse
			resp.Error = "ошибка сканирования строки"
			respBytes, err := json.Marshal(resp)
			if err != nil {
				http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
				return
			}
			http.Error(res, string(respBytes), http.StatusInternalServerError)
			return
		}
		taskMap := map[string]string{
			"id":      strconv.FormatInt(id, 10),
			"date":    date,
			"title":   title,
			"comment": comment,
			"repeat":  repeat,
		}
		tasks = append(tasks, taskMap)
	}

	if tasks == nil {
		tasks = make([]map[string]string, 0)
	}
	resp, err := json.Marshal(map[string][]map[string]string{"tasks": tasks})
	if err != nil {
		var resp models.ErrorResponse
		resp.Error = "ошибка сериализации ответа"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(resp)
	if err != nil {
		var resp models.ErrorResponse
		resp.Error = "ошибка записи ответа"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) HandleGetTask(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")

	if id != "" {
		row := h.DB.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id)
		var task models.Task
		err := row.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			var resp models.ErrorResponse
			resp.Error = "задача с указанным id не найдена"
			respBytes, err := json.Marshal(resp)
			if err != nil {
				http.Error(res, "ошибка при сериализации ответа", http.StatusNotFound)
				return
			}
			http.Error(res, string(respBytes), http.StatusNotFound)
			return
		}

		resp, err := json.Marshal(task)
		if err != nil {
			var resp models.ErrorResponse
			resp.Error = "ошибка сериализации ответа"
			respBytes, err := json.Marshal(resp)
			if err != nil {
				http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
				return
			}
			http.Error(res, string(respBytes), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusOK)
		_, err = res.Write(resp)
		if err != nil {
			var resp models.ErrorResponse
			resp.Error = "ошибка записи ответа"
			respBytes, err := json.Marshal(resp)
			if err != nil {
				http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
				return
			}
			http.Error(res, string(respBytes), http.StatusInternalServerError)
			return
		}
	} else {
		var resp models.ErrorResponse
		resp.Error = "отсутствует обязательный параметр id"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
			return
		}
		http.Error(res, string(respBytes), http.StatusBadRequest)
		return
	}
}

func (h *Handlers) HandlePutTask(res http.ResponseWriter, req *http.Request) {
	var taskUpdates map[string]interface{}
	err := json.NewDecoder(req.Body).Decode(&taskUpdates)
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

	id, ok := taskUpdates["id"].(string)
	if !ok {
		var resp models.ErrorResponse
		resp.Error = "отсутствует обязательное поле id"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
			return
		}
		http.Error(res, string(respBytes), http.StatusBadRequest)
		return
	}

	title, ok := taskUpdates["title"].(string)
	if !ok || title == "" {
		var resp models.ErrorResponse
		resp.Error = "отсутствует обязательное поле title"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
			return
		}
		http.Error(res, string(respBytes), http.StatusBadRequest)
		return
	}

	date, ok := taskUpdates["date"].(string)
	if !ok || date == "" {
		var resp models.ErrorResponse
		resp.Error = "отсутствует обязательное поле date"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
			return
		}
		http.Error(res, string(respBytes), http.StatusBadRequest)
		return
	}

	_, err = time.Parse("20060102", date)
	if err != nil {
		var resp models.ErrorResponse
		resp.Error = "недопустимый формат date"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
			return
		}
		http.Error(res, string(respBytes), http.StatusBadRequest)
		return
	}

	if repeat, ok := taskUpdates["repeat"].(string); ok {
		if repeat != "" {
			repeatParts := strings.SplitN(repeat, " ", 2)
			repeatType := repeatParts[0]
			if repeatType != "d" && repeatType != "w" && repeatType != "m" && repeatType != "y" {
				var resp models.ErrorResponse
				resp.Error = "недопустимый символ"
				respBytes, err := json.Marshal(resp)
				if err != nil {
					http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
					return
				}
				http.Error(res, string(respBytes), http.StatusBadRequest)
				return
			}
		}
	}

	query := "UPDATE scheduler SET "
	args := []interface{}{}
	i := 0

	for key, value := range taskUpdates {
		if key != "id" {
			if i > 0 {
				query += ", "
			}
			query += fmt.Sprintf("%s = ?", key)
			args = append(args, value)
			i++
		}
	}

	query += " WHERE id = ?"
	args = append(args, id)

	result, err := h.DB.Exec(query, args...)
	if err != nil {
		var resp models.ErrorResponse
		resp.Error = "ошибка запроса к базе данных"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		var resp models.ErrorResponse
		resp.Error = "ошибка получения числа затронутых строк"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		var resp models.ErrorResponse
		resp.Error = "задача с указанным id не найдена"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusNotFound)
			return
		}
		http.Error(res, string(respBytes), http.StatusNotFound)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write([]byte(`{}`))
	if err != nil {
		var resp models.ErrorResponse
		resp.Error = "ошибка при записи ответа"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
	}
}

func (h *Handlers) HandleDeleteTask(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")

	if id == "" {
		var resp models.ErrorResponse
		resp.Error = "не передан идентификатор"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
			return
		}
		http.Error(res, string(respBytes), http.StatusBadRequest)
		return
	}

	sqlRes, err := h.DB.Exec("DELETE FROM scheduler WHERE id = ?", id)
	if err != nil {
		var resp models.ErrorResponse
		resp.Error = "ошибка запроса к базе данных"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := sqlRes.RowsAffected()
	if err != nil {
		var resp models.ErrorResponse
		resp.Error = "ошибка получения количества затронутых строк"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		var resp models.ErrorResponse
		resp.Error = "задача с указанным id не найдена"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusNotFound)
			return
		}
		http.Error(res, string(respBytes), http.StatusNotFound)
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
		var resp models.ErrorResponse
		resp.Error = "ошибка при записи ответа"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
	}
}

func (h *Handlers) HandleDoneTask(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")

	if id == "" {
		var resp models.ErrorResponse
		resp.Error = "не передан идентификатор"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
			return
		}
		http.Error(res, string(respBytes), http.StatusBadRequest)
		return
	}

	row := h.DB.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id)
	var taskID int64
	var date, title, comment, repeat string
	err := row.Scan(&taskID, &date, &title, &comment, &repeat)
	if err != nil {
		if err == sql.ErrNoRows {
			var resp models.ErrorResponse
			resp.Error = "задача с указанным id не найдена"
			respBytes, err := json.Marshal(resp)
			if err != nil {
				http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
				return
			}
			http.Error(res, string(respBytes), http.StatusNotFound)
			return
		}
		var resp models.ErrorResponse
		resp.Error = "ошибка сканирования строки"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
		return
	}

	if repeat == "" {
		_, err = h.DB.Exec("DELETE FROM scheduler WHERE id = ?", taskID)
		if err != nil {
			var resp models.ErrorResponse
			resp.Error = "ошибка запроса к базе данных"
			respBytes, err := json.Marshal(resp)
			if err != nil {
				http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
				return
			}
			http.Error(res, string(respBytes), http.StatusInternalServerError)
			return
		}
	} else {
		parsedDate, err := time.Parse("20060102", date)
		if err != nil {
			var resp models.ErrorResponse
			resp.Error = "недопустимый формат date"
			respBytes, err := json.Marshal(resp)
			if err != nil {
				http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
				return
			}
			http.Error(res, string(respBytes), http.StatusBadRequest)
			return
		}
		if parsedDate.Format("20060102") == time.Now().Format("20060102") {
			parsedDate = parsedDate.AddDate(0, 0, -1)
			date, err = h.NextDate(parsedDate, date, repeat)
		} else {
			date, err = h.NextDate(time.Now(), date, repeat)
		}
		if err != nil {
			var resp models.ErrorResponse
			resp.Error = "правило повторения указано в неправильном формате"
			respBytes, err := json.Marshal(resp)
			if err != nil {
				http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
				return
			}
			http.Error(res, string(respBytes), http.StatusBadRequest)
			return
		}
		_, err = h.DB.Exec("UPDATE scheduler SET date = ? WHERE id = ?", date, taskID)
		if err != nil {
			var resp models.ErrorResponse
			resp.Error = "ошибка запроса к базе данных"
			respBytes, err := json.Marshal(resp)
			if err != nil {
				http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
				return
			}
			http.Error(res, string(respBytes), http.StatusInternalServerError)
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
		var resp models.ErrorResponse
		resp.Error = "ошибка при записи ответа"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
	}
}