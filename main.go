package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	r := chi.NewRouter()

	PORT := os.Getenv("TODO_PART")

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

	if err := http.ListenAndServe(fmt.Sprintf(":%s", PORT), r); err != nil {
		fmt.Printf("Ошибка при запуске сервера: %s", err.Error())
		return
	}
}
