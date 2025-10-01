package user

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"redditclone/pkg/generator"
	"redditclone/pkg/session"
)

type ServiceInterface interface {
	Register(username, password string) (*User, error)
	Login(username, password string) (*User, error)
}

type Service struct {
	Repo    Repository
	Session session.Repository
}

func NewService(repo Repository, session session.Repository) *Service {
	return &Service{Repo: repo, Session: session}
}

func (s *Service) Register(username, password string) (*User, error) {
	exist, err := s.Repo.FindByUsername(username)
	if exist != nil && err == nil {
		return nil, errors.New("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password error: %s", err)
	}

	userID, err := generator.GenerateRandomID(24)
	if err != nil {
		return nil, fmt.Errorf("UserID gen error: %s", err)
	}

	user := &User{
		ID:       userID,
		Username: username,
		Password: string(hashedPassword),
	}

	err = s.Repo.Create(user)
	if err != nil {
		return nil, err
	}

	sessionID, err := generator.GenerateRandomID(24)
	if err != nil {
		return nil, fmt.Errorf("SessionID gen error: %s", err)
	}
	_, err = s.Session.Create(user.ID, sessionID)
	if err != nil {
		return nil, errors.New("failed to create session")
	}

	return user, nil
}

func (s *Service) Login(username, password string) (*User, error) {
	user, err := s.Repo.FindByUsername(username)
	if err != nil {
		return nil, errors.New("user not found")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	sessionID, err := generator.GenerateRandomID(24)
	if err != nil {
		return nil, fmt.Errorf("SessionID gen error: %s", err)
	}
	_, err = s.Session.Create(user.ID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %s", err)
	}

	return user, nil
}
