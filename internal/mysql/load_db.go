package mysql

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func LoadDB() *sql.DB {
	db, err := sql.Open("mysql", os.Getenv("MYSQL_DSN"))
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal("Cannot connect to DB:", err)
	}
	if err := exec(db); err != nil {
		log.Fatal("Cannot create tables:", err)
	}
	return db
}

func exec(db *sql.DB) error {
	files := []string{
		"./internal/mysql/users.sql",
		"./internal/mysql/sessions.sql",
	}
	for _, file := range files {
		query, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}
		if _, err := db.Exec(string(query)); err != nil {
			return fmt.Errorf("failed to execute %s: %w", file, err)
		}
	}
	return nil
}
