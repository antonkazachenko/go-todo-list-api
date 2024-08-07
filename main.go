package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/antonkazachenko/go-todo-list-api/config"
	"github.com/antonkazachenko/go-todo-list-api/internal/service"
	storage "github.com/antonkazachenko/go-todo-list-api/internal/storage/sqlite"
	"github.com/antonkazachenko/go-todo-list-api/routes"
)

func main() {
	db := storage.InitDB()
	defer db.Close()

	taskRepo := storage.NewSQLiteTaskRepository(db)

	taskService := service.NewTaskService(taskRepo)

	router := routes.RegisterRoutes(taskService)

	fileServer := http.FileServer(http.Dir("./web"))
	router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".css" {
			w.Header().Set("Content-Type", "text/css")
		}
		fileServer.ServeHTTP(w, r)
	})

	address := fmt.Sprintf(":%s", config.TODO_PORT)
	log.Printf("Starting server on %s", address)
	if err := http.ListenAndServe(address, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
