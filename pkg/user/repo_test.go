package user_test

import (
	"database/sql"
	"testing"

	"redditclone/pkg/user"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)

	schema := `
	CREATE TABLE users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL
	);`

	_, err = db.Exec(schema)
	assert.NoError(t, err)

	return db
}

func setupTestBadDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)

	schema := `
	DROP TABLE IF EXISTS users;
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			password TEXT NOT NULL
	);`

	_, err = db.Exec(schema)
	assert.NoError(t, err)

	return db
}

func TestMySQLRepo_CreateAndFind(t *testing.T) {
	db := setupTestDB(t)
	repo := user.NewMySQLRepo(db)

	_user_ := &user.User{
		ID:       "user123",
		Username: "sj379d0xmsdl028sfdy3",
		Password: "hashed_pass",
	}
	err := repo.Create(_user_)
	assert.NoError(t, err)

	_user2_ := &user.User{
		ID:       "user123", // same id
		Username: "sj379d0xmsdl028sfdy3",
		Password: "hashed_pass",
	}
	err = repo.Create(_user2_)
	assert.Error(t, err)

	// Test FindByUsername
	u, err := repo.FindByUsername(_user_.Username)
	assert.NoError(t, err)
	assert.NotNil(t, u)

	_user3_ := &user.User{
		ID:       "user1230",
		Username: "sj379d0xm9sdl028sfdy3",
		Password: "hashed_pass",
	}
	u2, err := repo.FindByUsername(_user3_.Username)
	assert.Error(t, err)
	assert.Nil(t, u2)
	assert.Equal(t, "user not found", err.Error())

	db2 := setupTestBadDB(t)
	repo2 := user.NewMySQLRepo(db2)

	_, err = db2.Exec("INSERT INTO users (id, password) VALUES (?, ?)", "u123", "somepass")
	assert.NoError(t, err)

	_, err = repo2.FindByUsername("whoever")
	assert.Error(t, err)

	assert.NotEqual(t, "user not found", err.Error())
}
