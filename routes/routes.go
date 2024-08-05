package routes

import (
	"database/sql"

	"github.com/antonkazachenko/go-todo-list-api/handlers"
	"github.com/antonkazachenko/go-todo-list-api/middleware"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(db *sql.DB) *chi.Mux {
	r := chi.NewRouter()
	h := &handlers.Handlers{DB: db}

	r.Get("/api/nextdate", h.HandleNextDate)
	r.Post("/api/task", middleware.Auth(h.HandleAddTask))
	r.Get("/api/tasks", middleware.Auth(h.HandleGetTasks))
	r.Get("/api/task", middleware.Auth(h.HandleGetTask))
	r.Put("/api/task", middleware.Auth(h.HandlePutTask))
	r.Delete("/api/task", middleware.Auth(h.HandleDeleteTask))
	r.Post("/api/task/done", middleware.Auth(h.HandleDoneTask))
	r.Post("/api/signin", h.HandleSignIn)

	return r
}
