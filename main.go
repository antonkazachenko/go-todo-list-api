package main

import (
	"fmt"
	"github.com/antonkazachenko/go-todo-list-api/routes"
	"log"
	"net/http"
	"path/filepath"

	"github.com/antonkazachenko/go-todo-list-api/config"
	"github.com/antonkazachenko/go-todo-list-api/database"
)

func main() {
	db := database.InitDB()
	defer db.Close()

	r := routes.RegisterRoutes(db)

	fileServer := http.FileServer(http.Dir("./web"))
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".css" {
			w.Header().Set("Content-Type", "text/css")
		}
		fileServer.ServeHTTP(w, r)
	})

	if err := http.ListenAndServe(fmt.Sprintf(":%s", config.TODO_PORT), r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
