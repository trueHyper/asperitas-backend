package session

import (
	"database/sql"
	"time"
)

type MySQLSessionRepo struct {
	DB *sql.DB
}

func NewMySQLSessionRepo(db *sql.DB) *MySQLSessionRepo {
	return &MySQLSessionRepo{DB: db}
}

func (r *MySQLSessionRepo) Create(userID string, sessionID string) (string, error) {
	_, err := r.DB.Exec(`
		INSERT INTO sessions (id, user_id, created_at, expires_at)
		VALUES (?, ?, ?, ?)
	`, sessionID, userID, time.Now(), time.Now().Add(time.Hour*1))

	return sessionID, err
}

func (r *MySQLSessionRepo) IsValid(userID string) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sessions 
			WHERE user_id = ? AND expires_at > ?
		)
	`, userID, time.Now().UTC()).Scan(&exists)
	return exists, err
}

func (r *MySQLSessionRepo) Invalidate(userID string) error {
	_, err := r.DB.Exec(`
		DELETE FROM sessions WHERE user_id = ?
	`, userID)
	return err
}
