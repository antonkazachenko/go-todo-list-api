package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3" // import the sqlite3 driver
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// TODO: что делать если дата начала позже значения now
func NextDate(now time.Time, date string, repeat string) (string, error) {
	parsedDate, err := time.Parse("20060102", date)
	if err != nil {
		return _, errors.New("недопустимый формат date")
	}

	repeatParts := strings.SplitN(repeat, " ", 2)
	repeatType, repeatRule := repeatParts[0], repeatParts[1]
	if len(repeatParts) < 2 {
		repeatRule = ""
	}

	if repeatType == "d" {
		if repeatRule == "" {
			return _, errors.New("не указан интервал в днях")
		} else {
			numberOfDay, err := strconv.Atoi(repeatRule)
			if err != nil {
				return _, errors.New("некорректно указано правило repeat")
			}
			for now.After(parsedDate) {
				parsedDate.AddDate(0, 0, numberOfDay)
			}
		}
	} else if repeatType == "y" {
		for now.After(parsedDate) {
			parsedDate.AddDate(1, 0, 0)
		}
	} else {
		return _, errors.New("недопустимый символ")
	}

	return parsedDate.String(), nil
}

func handleNextDate(res http.ResponseWriter, req *http.Request) {
	now, err := time.Parse("20060102", chi.URLParam(req, "now"))

	if err != nil {
		http.Error(res, "Неправильный формат парамертра now", http.StatusBadRequest)
		return
	}

	date := chi.URLParam(req, "date")
	repeat := chi.URLParam(req, "repeat")

	newDate, err := NextDate(now, date, repeat)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: узнать что имеется ввиду под удалением из дб, у меня же нет никакого айди таска

	resp, err := json.Marshal(newDate)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(resp)
	if err != nil {
		fmt.Printf("Error in writing a response for /api/nextdate GET request,\n %v", err)
		return
	}
}

func main() {
	r := chi.NewRouter()

	TODO_DBFILE := os.Getenv("TODO_DBFILE")
	if TODO_DBFILE == "" {
		TODO_DBFILE = "scheduler.db"
	}

	db, err := sql.Open("sqlite3", TODO_DBFILE)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Always attempt to create the table and index if they do not exist
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS scheduler (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL,
		title TEXT NOT NULL,
		comment TEXT NOT NULL,
		repeat TEXT NOT NULL
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

	r.Get("/api/nextdate")

	if err := http.ListenAndServe(fmt.Sprintf(":%s", PORT), r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
