package user_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"redditclone/pkg/user"
)

type mockRepo struct {
	mock.Mock
}

type mockSession struct {
	mock.Mock
}

func (m *mockRepo) FindByUsername(username string) (*user.User, error) {
	args := m.Called(username)
	if u := args.Get(0); u != nil {
		return u.(*user.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepo) Create(u *user.User) error {
	return m.Called(u).Error(0)
}

func (m *mockSession) Create(userID, sessionID string) (string, error) {
	args := m.Called(userID, sessionID)
	return args.String(0), args.Error(1)
}

func (m *mockSession) IsValid(userID string) (bool, error) {
	args := m.Called(userID)
	return args.Bool(0), args.Error(1)
}

func (m *mockSession) Invalidate(userID string) error {
	return m.Called(userID).Error(0)
}

func TestService_Register(t *testing.T) {
	repo := new(mockRepo)
	session := new(mockSession)
	svc := user.NewService(repo, session)

	t.Run("success", func(t *testing.T) {
		repo.On("FindByUsername", "newuser").Return(nil, nil)
		repo.On("Create", mock.AnythingOfType("*user.User")).Return(nil)
		session.On("Create", mock.Anything, mock.Anything).Return("sessid", nil)

		u, err := svc.Register("newuser", "securepass")

		assert.NoError(t, err)
		assert.NotNil(t, u)
		assert.Equal(t, "newuser", u.Username)
	})

	t.Run("user already exists", func(t *testing.T) {
		repo.On("FindByUsername", "existing").Return(&user.User{Username: "existing"}, nil)

		u, err := svc.Register("existing", "pass")

		assert.Error(t, err)
		assert.Nil(t, u)
		assert.Equal(t, "user already exists", err.Error())
	})
}

func TestService_Login(t *testing.T) {
	repo := new(mockRepo)
	session := new(mockSession)
	svc := user.NewService(repo, session)

	hashed, err := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		repo.On("FindByUsername", "valid").Return(&user.User{
			ID:       "uid",
			Username: "valid",
			Password: string(hashed),
		}, nil)
		session.On("Create", "uid", mock.Anything).Return("sessid", nil)

		u, err := svc.Login("valid", "correct")

		assert.NoError(t, err)
		assert.Equal(t, "valid", u.Username)
	})

	t.Run("not found", func(t *testing.T) {
		repo.On("FindByUsername", "ghost").Return(nil, errors.New("not found"))

		u, err := svc.Login("ghost", "any")

		assert.Error(t, err)
		assert.Nil(t, u)
		assert.Equal(t, "user not found", err.Error())
	})

	t.Run("wrong password", func(t *testing.T) {
		repo.On("FindByUsername", "valid").Return(&user.User{
			ID:       "uid",
			Username: "valid",
			Password: string(hashed),
		}, nil)

		u, err := svc.Login("valid", "wrong")

		assert.Error(t, err)
		assert.Nil(t, u)
		assert.Equal(t, "invalid credentials", err.Error())
	})

	t.Run("Hashing pass error", func(t *testing.T) {
		repo.On("FindByUsername", "valid").Return(&user.User{
			ID:       "uid",
			Username: "valid",
			Password: "oops",
		}, nil)

		u, err := svc.Login("valid", "wrong")

		assert.Error(t, err)
		assert.Nil(t, u)
		assert.Equal(t, "invalid credentials", err.Error())
	})
}
