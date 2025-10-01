package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"redditclone/pkg/user"

	jwt "github.com/dgrijalva/jwt-go"
)

type LoginForm struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Handler struct {
	Service user.ServiceInterface
	Logger  *slog.Logger
}

type FieldError struct {
	Location string `json:"location"`
	Param    string `json:"param"`
	Value    string `json:"value"`
	Msg      string `json:"msg"`
}

func NewUserHandler(service user.ServiceInterface, logger *slog.Logger) *Handler {
	return &Handler{
		Service: service,
		Logger:  logger,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req LoginForm
	if ok := DecodeJSONBody(w, r, &req); !ok {
		return
	}

	user, err := h.Service.Register(req.Username, req.Password)
	if err != nil {
		if err.Error() != "user already exists" {
			h.Logger.Error("register", "error", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ok := WriteResp(w, h.Logger, map[string]any{
			"errors": []FieldError{
				{
					Location: "body",
					Param:    "username",
					Value:    req.Username,
					Msg:      "already exists",
				},
			},
		}, http.StatusUnprocessableEntity); ok {
			h.Logger.Error("register", "error", err.Error(), "user", user)
		}
	} else {
		GenerateToken(user.Username, user.ID, w, h.Logger, "register")
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginForm
	if ok := DecodeJSONBody(w, r, &req); !ok {
		return
	}

	user, err := h.Service.Login(req.Username, req.Password)
	if err != nil {
		var msg string
		if err.Error() == "user not found" {
			msg = "user not found"
		} else {
			msg = "invalid password"
		}
		if ok := WriteResp(w, h.Logger, map[string]any{"message": msg}, http.StatusUnauthorized); ok {
			h.Logger.Error("login", "error", "unauthorized", "user", user)
		}
	} else {
		GenerateToken(user.Username, user.ID, w, h.Logger, "login")
	}
}

func DecodeJSONBody(w http.ResponseWriter, r *http.Request, req any) bool {
	if r.Header.Get("Content-Type") != "application/json" {
		writeError(w, http.StatusBadRequest, typeError, "invalid Content-Type")
		return false
	}

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		writeError(w, http.StatusBadRequest, typeError, "bad json")
		return false
	}

	return true
}

func GenerateToken(username, userID string, w http.ResponseWriter, logger *slog.Logger, action string) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": map[string]string{
			"username": username,
			"id":       userID,
		},
		"iat": time.Now().UTC().Unix(),
		"exp": time.Now().Add(time.Hour * 1).UTC().Unix(),
	})
	JWTSecret := os.Getenv("JWT_SECRET")
	tokenString, err := token.SignedString([]byte(JWTSecret))
	if err != nil {
		logger.Error("token signing", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if ok := WriteResp(w, logger, map[string]any{"token": tokenString}, http.StatusOK); ok {
		logger.Info(action, "user", userID)
	}
}

func WriteResp(w http.ResponseWriter, logger *slog.Logger, body map[string]any, status int) bool {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		logger.Error("failed to write JSON response", slog.Any("err", err))
		return false
	}
	return true
}
