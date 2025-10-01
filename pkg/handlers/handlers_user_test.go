package handlers_test

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"redditclone/pkg/handlers"
	"redditclone/pkg/user"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) Register(username, password string) (*user.User, error) {
	args := m.Called(username, password)
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *mockService) Login(username, password string) (*user.User, error) {
	args := m.Called(username, password)
	return args.Get(0).(*user.User), args.Error(1)
}

func TestLoginHandler(t *testing.T) {
	m := new(mockService)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	m.On("Login", "validuser", "correct").Return(&user.User{ID: "id", Username: "validuser"}, nil)
	m.On("Login", "wronguser", "correct").Return((*user.User)(nil), errors.New("user not found"))
	m.On("Login", "validuser", "wrong").Return((*user.User)(nil), errors.New("invalid credentials"))

	handler := handlers.NewUserHandler(m, logger)

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Successful login",
			body:           `{"username":"validuser","password":"correct"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "User not found",
			body:           `{"username":"wronguser","password":"correct"}`,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "user not found",
		},
		{
			name:           "Invalid credentials",
			body:           `{"username":"validuser","password":"wrong"}`,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid password",
		},
		{
			name:           "Bad Content-Type",
			body:           `{"username":"validuser","password":"wrong"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  `{"error":"invalid Content-Type"}`,
		},
		{
			name:           "Bad JSON",
			body:           `{"username" oops "validuser","password":"wrong"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  `{"error":"bad json"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(test.body))
			if test.name == "Bad Content-Type" {
				req.Header.Set("Content-Type", "plain/text")
			} else {
				req.Header.Set("Content-Type", "application/json")
			}

			rr := httptest.NewRecorder()

			handler.Login(rr, req)

			assert.Equal(t, test.expectedStatus, rr.Code)

			if test.expectedError != "" {
				assert.Contains(t, rr.Body.String(), test.expectedError)
			}
		})
	}

	m.AssertExpectations(t)
}

func TestRegister(t *testing.T) {
	m := new(mockService)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	m.On("Register", "validuser", "correct").Return(&user.User{ID: "id", Username: "validuser"}, nil)
	m.On("Register", "existinguser", "password").Return((*user.User)(nil), errors.New("user already exists"))
	m.On("Register", "wronguser", "password").Return((*user.User)(nil), errors.New("unexpected error"))

	handler := handlers.NewUserHandler(m, logger)

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Successful registration",
			body:           `{"username":"validuser","password":"correct"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "User already exists",
			body:           `{"username":"existinguser","password":"password"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError:  "already exists",
		},
		{
			name:           "Unexpected error",
			body:           `{"username":"wronguser","password":"password"}`,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "unexpected error",
		},
		{
			name:           "Bad Content-Type",
			body:           `{"username":"validuser","password":"correct"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  `invalid Content-Type`,
		},
		{
			name:           "Bad JSON",
			body:           `{"username" oops "validuser","password":"correct"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  `bad json`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(test.body))
			if test.name == "Bad Content-Type" {
				req.Header.Set("Content-Type", "plain/text")
			} else {
				req.Header.Set("Content-Type", "application/json")
			}

			rr := httptest.NewRecorder()

			handler.Register(rr, req)

			assert.Equal(t, test.expectedStatus, rr.Code)

			if test.expectedError != "" {
				assert.Contains(t, rr.Body.String(), test.expectedError)
			}
		})
	}

	m.AssertExpectations(t)
}

/* не работает
func TestLoginGenerateToken(t *testing.T) {

	//original := os.Getenv("JWT_SECRET")
	os.Unsetenv("JWT_SECRET")

	m := new(mockService)
	m.On("Login", "validuser", "correct").Return(&user.User{ID: "id", Username: "validuser"}, nil)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	handler := &handlers.Handler{
		Service: m,
		Logger:  logger,
	}


	t.Run("Token signing error", func(t *testing.T) {
		//os.Unsetenv("JWT_SECRET")

		req := httptest.NewRequest("POST", "/login", strings.NewReader(`{"username":"validuser","password":"correct"}`))
		req.Header.Set("Content-Type", "application/json")

		resp := httptest.NewRecorder()

		handler.Login(resp, req)

		if resp.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", resp.Code)
		}

		if !strings.Contains(resp.Body.String(), "token signing") {
			t.Errorf("expected error 'token signing' in response, got %s", resp.Body.String())
		}
	})
}
*/
