package middleware

import (
	"github.com/antonkazachenko/go-todo-list-api/config"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
)

func Auth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pass := config.TODO_PASS
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
