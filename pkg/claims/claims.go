package claims

import jwt "github.com/dgrijalva/jwt-go"

type contextKey string

const (
	TokenContextKey contextKey = "token"
)

type Claims struct {
	User struct {
		Username string `json:"username"`
		ID       string `json:"id"`
	} `json:"user"`
	jwt.StandardClaims
}
