package user

import (
	"database/sql"
	"errors"
)

type MySQLRepo struct {
	DB *sql.DB
}

func NewMySQLRepo(db *sql.DB) *MySQLRepo {
	return &MySQLRepo{DB: db}
}

func (r *MySQLRepo) Create(user *User) error {
	_, err := r.DB.Exec(
		"INSERT INTO users (id, username, password) VALUES (?, ?, ?)",
		user.ID, user.Username, user.Password,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *MySQLRepo) FindByUsername(username string) (*User, error) {
	var u User
	err := r.DB.QueryRow(
		"SELECT id, username, password FROM users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.Username, &u.Password)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &u, nil
}
