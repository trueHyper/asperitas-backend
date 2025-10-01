package middleware

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"redditclone/pkg/claims"
	"redditclone/pkg/session"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
)

const category string = "music|funny|videos|programming|news|fashion"

var (
	noSessUrls = map[string]string{
		"/api/login":                       http.MethodPost,
		"/api/register":                    http.MethodPost,
		"/api":                             http.MethodGet,
		"/api/posts/":                      http.MethodGet,
		"/api/post/{post_id:[a-zA-Z0-9]+}": http.MethodGet,
		"/api/user/{login:[a-zA-Z0-9]+}":   http.MethodGet,
		"/api/posts/{category:(?:" + category + ")}": http.MethodGet,
	}
)

func CheckJWT(sessionStore *session.MySQLSessionRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := mux.CurrentRoute(r)
			template, err := route.GetPathTemplate()

			if err != nil {
				http.Error(w, "Route not found", http.StatusNotFound)
				return
			}

			if method, ok := noSessUrls[template]; ok && method == r.Method {
				next.ServeHTTP(w, r)
				return
			}

			auth := r.Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
				log.Println("1")
				http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(auth, "Bearer ")

			hashSecretGetter := func(token *jwt.Token) (interface{}, error) {
				method, ok := token.Method.(*jwt.SigningMethodHMAC)
				if !ok || method.Alg() != "HS256" {
					http.Error(w, "bad sign method", http.StatusUnauthorized)
					return nil, nil
				}
				JWTSecret := os.Getenv("JWT_SECRET")
				return []byte(JWTSecret), nil
			}

			_claims_ := &claims.Claims{}

			_token_, err := jwt.ParseWithClaims(token, _claims_, hashSecretGetter)
			if err != nil || !_token_.Valid || _claims_.User.Username == "" {
				log.Println("_token_.Valid", _token_.Valid)
				http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			ok, err := sessionStore.IsValid(_claims_.User.ID)
			if err != nil || !ok {
				log.Println("2", _claims_.User.ID, _claims_.User.Username)
				log.Println(ok, err)
				http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claims.TokenContextKey, _claims_)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
