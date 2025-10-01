package routing

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"

	"redditclone/pkg/handlers"
	"redditclone/pkg/post"
	"redditclone/pkg/session"
	"redditclone/pkg/user"
)

const (
	staticPath   = "./static"
	postCategory = "music|funny|videos|programming|news|fashion"
)

func InitRoutes(api *mux.Router, db *sql.DB, mongoDB *mongo.Database, logger *slog.Logger) {

	sessionRepo := session.NewMySQLSessionRepo(db)

	userService := user.NewService(user.NewMySQLRepo(db), sessionRepo)
	userHandler := handlers.NewUserHandler(userService, logger)

	postService := &post.PostService{Repo: post.NewMongoRepo(mongoDB)}
	postHandler := handlers.NewPostHandler(postService, logger)

	/* -+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+ */

	authRouter := api.PathPrefix("").Subrouter()
	postsRouter := api.PathPrefix("/posts").Subrouter()
	userRouter := api.PathPrefix("/user").Subrouter()
	postRouter := api.PathPrefix("/post").Subrouter()

	/* auth routers */
	authRouter.HandleFunc("/register", userHandler.Register).Methods("POST").Name("register")
	authRouter.HandleFunc("/login", userHandler.Login).Methods("POST").Name("login")

	/* posts routers */
	postsRouter.HandleFunc("", postHandler.CreatePost).Methods("POST")
	postsRouter.HandleFunc("/", postHandler.GetAllPosts).Methods("GET")
	postsRouter.HandleFunc("/{category:(?:"+postCategory+")}", postHandler.GetPostsByCategory).Methods("GET")

	/* user routers */
	userRouter.HandleFunc("/{login:[a-zA-Z0-9]+}", postHandler.GetPostsByUser).Methods("GET")

	/* posts routers */
	postRouter.HandleFunc("/{post_id:[a-zA-Z0-9]+}", postHandler.GetPostByID).Methods("GET")
	postRouter.HandleFunc("/{post_id:[a-zA-Z0-9]+}", postHandler.AddComment).Methods("POST")
	postRouter.HandleFunc("/{post_id:[a-zA-Z0-9]+}", postHandler.DeletePost).Methods("DELETE")
	postRouter.HandleFunc("/{post_id:[a-zA-Z0-9]+}/{comm_id:[a-zA-Z0-9]+}", postHandler.RemoveComment).Methods("DELETE")
	postRouter.HandleFunc("/{post_id:[a-zA-Z0-9]+}/{action:(?:upvote|downvote|unvote)}", postHandler.AddVote).Methods("GET")
}

func ServeStaticFiles(r *mux.Router) {
	fs := http.FileServer(http.Dir(staticPath))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
}

func ServeFallback(r *mux.Router, logger *slog.Logger) {
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/static/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("[]")); err != nil { 
				logger.Error("failed to write fallback JSON", slog.String("path", r.URL.Path), slog.Any("error", err))
			}
			return
		}
		http.ServeFile(w, r, "static/html/index.html")
	})
}

func StartServer(r *mux.Router) {
	fmt.Println("\n\033[32m", "The server is running on http://localhost:8082", "\033[0m")
	if err := http.ListenAndServe(":8082", r); err != nil {
		log.Fatal("Server failed:", err)
	}
}
