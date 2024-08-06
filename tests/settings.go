package tests

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var Port = 7540
var DBFile = "../scheduler.db"
var FullNextDate = true
var Search = true
var Token = generateTestToken()

func generateTestToken() string {
	pass := "test12345"

	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	tokenString, err := token.SignedString([]byte(pass))
	if err != nil {
		panic(err)
	}

	return tokenString
}
