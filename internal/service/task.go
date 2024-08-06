package service

import (
	"errors"
	"strconv"
	"strings"
	"time"

	storage "github.com/antonkazachenko/go-todo-list-api/internal/storage/sqlite"
)

const Format = "20060102"

type TaskService struct {
	Repo *storage.SQLiteTaskRepository
}

func NewTaskService(repo *storage.SQLiteTaskRepository) *TaskService {
	return &TaskService{Repo: repo}
}

func (s *TaskService) NextDate(now time.Time, date string, repeat string) (string, error) {
	parsedDate, err := time.Parse(Format, date)
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

			if now.Format(Format) != parsedDate.Format(Format) {
				if now.After(parsedDate) {
					for now.After(parsedDate) || now.Format(Format) == parsedDate.Format(Format) {
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

	return parsedDate.Format(Format), nil
}
