package main

import (
	"redditclone/internal/config"
	"redditclone/internal/logger"
	"redditclone/internal/mongo"
	"redditclone/internal/mysql"
	"redditclone/internal/routing"
	"redditclone/pkg/middleware"
	"redditclone/pkg/session"

	"github.com/gorilla/mux"
)

func main() {
	config.Load() // load env var from .env

	db := mysql.LoadDB()
	defer db.Close()

	mongoDB := mongo.LoadDB()

	logger := logger.Load()

	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	api.Use(middleware.Panic)
	api.Use(middleware.CheckJWT(session.NewMySQLSessionRepo(db)))

	routing.InitRoutes(api, db, mongoDB, logger)
	routing.ServeStaticFiles(r)
	routing.ServeFallback(r, logger)
	routing.StartServer(r) // start sever on localhost:8082
}
