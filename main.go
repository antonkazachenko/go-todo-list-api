package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v4"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var db *sql.DB

type authResponse struct {
	Token string `json:"token"`
}

type idResponse struct {
	ID int64 `json:"id"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type Task struct {
	Date    string `json:"date,omitempty"`
	Title   string `json:"title"`
	Comment string `json:"comment,omitempty"`
	Repeat  string `json:"repeat,omitempty"`
}

func NextDate(now time.Time, date string, repeat string) (string, error) {
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

			//if numberOfDays != 0 && now.After(parsedDate) && now.Format("20060102") != parsedDate.Format("20060102") {
			//	parsedDate = parsedDate.AddDate(0, 0, numberOfDays)
			//}
			if now.Format("20060102") != parsedDate.Format("20060102") {
				if numberOfDays == 1 {
					fmt.Println("here")
				}
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

func handleNextDate(res http.ResponseWriter, req *http.Request) {
	nowParam := req.URL.Query().Get("now")
	date := req.URL.Query().Get("date")
	repeat := req.URL.Query().Get("repeat")

	now, err := time.Parse("20060102", nowParam)

	if err != nil {
		http.Error(res, "Неправильный формат парамeтра now", http.StatusBadRequest)
		return
	}
	var newDate string
	if repeat != "" {
		newDate, err = NextDate(now, date, repeat)
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

func handleAddTask(res http.ResponseWriter, req *http.Request) {
	var buf bytes.Buffer
	var task Task

	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		var resp errorResponse
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
		var resp errorResponse
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
		var resp errorResponse
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
			var resp errorResponse
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
			if task.Repeat == "d 1" {
				fmt.Println("here")
			}
			task.Date, err = NextDate(time.Now(), task.Date, task.Repeat)
			if err != nil {
				var resp errorResponse
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

	result, err := db.Exec("INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)",
		task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		var resp errorResponse
		resp.Error = "ошибка запроса к базе данных"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
			return
		}
		http.Error(res, string(respBytes), http.StatusBadRequest)
		return
	}

	var ret idResponse
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

func handleGetTasks(res http.ResponseWriter, req *http.Request) {
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
	rows, err := db.Query(query, args...)
	if err != nil {
		var resp errorResponse
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
			var resp errorResponse
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
		var resp errorResponse
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
		var resp errorResponse
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

func handleGetTask(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")

	if id != "" {
		row := db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id)
		var id int64
		var task Task
		err := row.Scan(&id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			var resp errorResponse
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
			var resp errorResponse
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
			var resp errorResponse
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
		var resp errorResponse
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

func handlePutTask(res http.ResponseWriter, req *http.Request) {
	var taskUpdates map[string]interface{}
	err := json.NewDecoder(req.Body).Decode(&taskUpdates)
	if err != nil {
		var resp errorResponse
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
		var resp errorResponse
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
		var resp errorResponse
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
		var resp errorResponse
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
		var resp errorResponse
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
				var resp errorResponse
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

	result, err := db.Exec(query, args...)
	if err != nil {
		var resp errorResponse
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
		var resp errorResponse
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
		var resp errorResponse
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
		var resp errorResponse
		resp.Error = "ошибка при записи ответа"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
	}
}

func handleDoneTask(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")

	if id == "" {
		var resp errorResponse
		resp.Error = "не передан идентификатор"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
			return
		}
		http.Error(res, string(respBytes), http.StatusBadRequest)
		return
	}

	row := db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id)
	var taskID int64
	var date, title, comment, repeat string
	err := row.Scan(&taskID, &date, &title, &comment, &repeat)
	if err != nil {
		if err == sql.ErrNoRows {
			var resp errorResponse
			resp.Error = "задача с указанным id не найдена"
			respBytes, err := json.Marshal(resp)
			if err != nil {
				http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
				return
			}
			http.Error(res, string(respBytes), http.StatusNotFound)
			return
		}
		var resp errorResponse
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
		_, err = db.Exec("DELETE FROM scheduler WHERE id = ?", taskID)
		if err != nil {
			var resp errorResponse
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
		fmt.Printf("date: %v --- ", date)
		parsedDate, err := time.Parse("20060102", date)
		if err != nil {
			var resp errorResponse
			resp.Error = "недопустимый формат date"
			respBytes, err := json.Marshal(resp)
			if err != nil {
				http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
				return
			}
			http.Error(res, string(respBytes), http.StatusBadRequest)
			return
		}
		if time.Now().After(parsedDate) || time.Now().Format("20060102") == parsedDate.Format("20060102") {
			date, err = NextDate(time.Now(), date, repeat)
		} else {
			date, err = NextDate(parsedDate.AddDate(0, 0, 1), date, repeat)
		}
		fmt.Println("date after: ", date)
		if err != nil {
			var resp errorResponse
			resp.Error = "правило повторения указано в неправильном формате"
			respBytes, err := json.Marshal(resp)
			if err != nil {
				http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
				return
			}
			http.Error(res, string(respBytes), http.StatusBadRequest)
			return
		}
		_, err = db.Exec("UPDATE scheduler SET date = ? WHERE id = ?", date, taskID)
		if err != nil {
			var resp errorResponse
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
		var resp errorResponse
		resp.Error = "ошибка при записи ответа"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
	}
}

func handleDeleteTask(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")

	if id == "" {
		var resp errorResponse
		resp.Error = "не передан идентификатор"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusBadRequest)
			return
		}
		http.Error(res, string(respBytes), http.StatusBadRequest)
		return
	} else {
		sqlRes, err := db.Exec("DELETE FROM scheduler WHERE id = ?", id)
		if err != nil {
			var resp errorResponse
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
			var resp errorResponse
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
			var resp errorResponse
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
			var resp errorResponse
			resp.Error = "ошибка при записи ответа"
			respBytes, err := json.Marshal(resp)
			if err != nil {
				http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
				return
			}
			http.Error(res, string(respBytes), http.StatusInternalServerError)
		}
		return
	}
}

func handleSignIn(res http.ResponseWriter, req *http.Request) {
	pass := os.Getenv("TODO_PASSWORD")
	if len(pass) == 0 {
		var resp errorResponse
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
		var resp errorResponse
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
		var resp errorResponse
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
		var resp errorResponse
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
		var resp errorResponse
		resp.Error = "ошибка создания токена"
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(res, "ошибка при сериализации ответа", http.StatusInternalServerError)
			return
		}
		http.Error(res, string(respBytes), http.StatusInternalServerError)
		return
	}

	var resp authResponse
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

func auth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pass := os.Getenv("TODO_PASSWORD")
		if len(pass) > 0 {
			var jwtToken string
			cookie, err := r.Cookie("token")
			if err == nil {
				jwtToken = cookie.Value
			}

			var valid bool
			if jwtToken != "" {
				token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, http.ErrAbortHandler
					}
					return []byte(pass), nil
				})

				if err == nil && token.Valid {
					valid = true
				}
			}

			if !valid {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	})
}

func main() {
	r := chi.NewRouter()

	TODO_DBFILE := os.Getenv("TODO_DBFILE")
	if TODO_DBFILE == "" {
		TODO_DBFILE = "scheduler.db"
	}

	var err error
	db, err = sql.Open("sqlite3", TODO_DBFILE)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS scheduler (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL,
		title TEXT NOT NULL,
		comment TEXT,
		repeat TEXT
	)`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_scheduler_date ON scheduler (date)`)
	if err != nil {
		log.Fatalf("Failed to create index: %v", err)
	}

	PORT := os.Getenv("TODO_PORT")
	if PORT == "" {
		PORT = "7540"
	}

	fileServer := http.FileServer(http.Dir("./web"))
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".css" {
			w.Header().Set("Content-Type", "text/css")
		}
		fileServer.ServeHTTP(w, r)
	})

	r.Get("/api/nextdate", handleNextDate)
	r.Post("/api/task", auth(handleAddTask))
	r.Get("/api/tasks", auth(handleGetTasks))
	r.Get("/api/task", auth(handleGetTask))
	r.Put("/api/task", auth(handlePutTask))
	r.Delete("/api/task", auth(handleDeleteTask))
	r.Post("/api/task/done", auth(handleDoneTask))
	r.Post("/api/signin", handleSignIn)

	if err := http.ListenAndServe(fmt.Sprintf(":%s", PORT), r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
