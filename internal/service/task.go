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

	repeatType, repeatRule := parseRepeatRule(repeat)

	switch repeatType {
	case "d":
		return calculateDailyRepeat(now, parsedDate, repeatRule)
	case "y":
		return calculateYearlyRepeat(now, parsedDate)
	case "w":
		return calculateWeeklyRepeat(now, parsedDate, repeatRule)
	case "m":
		return calculateMonthlyRepeat(now, parsedDate, repeatRule)
	default:
		return "", errors.New("недопустимый символ")
	}
}

func parseRepeatRule(repeat string) (string, string) {
	repeatParts := strings.SplitN(repeat, " ", 2)
	repeatType := ""
	repeatRule := ""

	if len(repeatParts) > 0 {
		repeatType = repeatParts[0]
	}
	if len(repeatParts) > 1 {
		repeatRule = repeatParts[1]
	}

	return repeatType, repeatRule
}

func calculateDailyRepeat(now, parsedDate time.Time, repeatRule string) (string, error) {
	if repeatRule == "" {
		return "", errors.New("не указан интервал в днях")
	}

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

	return parsedDate.Format(Format), nil
}

func calculateYearlyRepeat(now, parsedDate time.Time) (string, error) {
	parsedDate = parsedDate.AddDate(1, 0, 0)
	for now.After(parsedDate) {
		parsedDate = parsedDate.AddDate(1, 0, 0)
	}
	return parsedDate.Format(Format), nil
}

func calculateWeeklyRepeat(now, parsedDate time.Time, repeatRule string) (string, error) {
	daysOfWeek, err := parseDaysOfWeek(repeatRule)
	if err != nil {
		return "", err
	}

	if now.Before(parsedDate) {
		for {
			parsedDate = parsedDate.AddDate(0, 0, 1)
			if daysOfWeek[int(parsedDate.Weekday())] {
				break
			}
		}
	} else {
		for {
			if daysOfWeek[int(parsedDate.Weekday())] {
				if now.Before(parsedDate) {
					break
				}
			}
			parsedDate = parsedDate.AddDate(0, 0, 1)
		}
	}

	return parsedDate.Format(Format), nil
}

func parseDaysOfWeek(repeatRule string) (map[int]bool, error) {
	daysOfWeek := make(map[int]bool)
	substrings := strings.Split(repeatRule, ",")

	for _, value := range substrings {
		number, err := strconv.Atoi(value)
		if err != nil {
			return nil, errors.New("ошибка конвертации значения дня недели")
		}
		if number < 1 || number > 7 {
			return nil, errors.New("недопустимое значение дня недели")
		}
		if number == 7 {
			number = 0
		}
		daysOfWeek[number] = true
	}
	return daysOfWeek, nil
}

func calculateMonthlyRepeat(now, parsedDate time.Time, repeatRule string) (string, error) {
	daysPart, monthsPart := splitMonthRule(repeatRule)
	dayMap, err := parseDays(daysPart)
	if err != nil {
		return "", err
	}
	monthMap, err := parseMonths(monthsPart)
	if err != nil {
		return "", err
	}

	if now.Before(parsedDate) {
		for {
			parsedDate = parsedDate.AddDate(0, 0, 1)
			if isValidDateForMonthlyRepeat(parsedDate, dayMap, monthMap) {
				break
			}
		}
	} else {
		for {
			if isValidDateForMonthlyRepeat(parsedDate, dayMap, monthMap) {
				if now.Before(parsedDate) {
					break
				}
			}
			parsedDate = parsedDate.AddDate(0, 0, 1)
		}
	}

	return parsedDate.Format(Format), nil
}

func isValidDateForMonthlyRepeat(parsedDate time.Time, dayMap map[int]bool, monthMap map[int]bool) bool {
	month := int(parsedDate.Month())

	if len(monthMap) > 0 && !monthMap[month] {
		return false
	}

	lastDayOfMonth := time.Date(parsedDate.Year(), parsedDate.Month()+1, 0, 0, 0, 0, 0, parsedDate.Location()).Day()
	for targetDay := range dayMap {
		if targetDay > 0 {
			if parsedDate.Day() == targetDay {
				return true
			}
		} else if targetDay < 0 {
			if parsedDate.Day() == lastDayOfMonth+targetDay+1 {
				return true
			}
		}
	}

	return false
}

func splitMonthRule(repeatRule string) (string, string) {
	repeatParts := strings.Split(repeatRule, " ")
	daysPart := repeatParts[0]
	monthsPart := ""
	if len(repeatParts) > 1 {
		monthsPart = repeatParts[1]
	}
	return daysPart, monthsPart
}

func parseDays(daysPart string) (map[int]bool, error) {
	dayMap := make(map[int]bool)
	days := strings.Split(daysPart, ",")

	for _, dayStr := range days {
		day, err := strconv.Atoi(dayStr)
		if err != nil {
			return nil, errors.New("ошибка конвертации значения дня месяца")
		}
		if day < -2 || day > 31 || day == 0 {
			return nil, errors.New("недопустимое значение дня месяца")
		}
		dayMap[day] = true
	}
	return dayMap, nil
}

func parseMonths(monthsPart string) (map[int]bool, error) {
	monthMap := make(map[int]bool)
	months := strings.Split(monthsPart, ",")

	for _, monthStr := range months {
		if monthStr != "" {
			month, err := strconv.Atoi(monthStr)
			if err != nil {
				return nil, errors.New("ошибка конвертации значения месяца")
			}
			if month < 1 || month > 12 {
				return nil, errors.New("недопустимое значение месяца")
			}
			monthMap[month] = true
		}
	}
	return monthMap, nil
}
