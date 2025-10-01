package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"redditclone/pkg/claims"
	"redditclone/pkg/post"
)

const (
	lenID          int    = 24
	typeError      string = "error"
	typeMessage    string = "message"
	muxVarPostID   string = "post_id"
	muxVarCommID   string = "comm_id"
	muxVarAction   string = "action"
	muxVarLogin    string = "login"
	muxVarCategory string = "category"
)

type PostHandler struct {
	Service post.ServicePost
	Logger  *slog.Logger
}

func NewPostHandler(service post.ServicePost, logger *slog.Logger) *PostHandler {
	return &PostHandler{
		Service: service,
		Logger:  logger,
	}
}

func (h *PostHandler) GetAllPosts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.Logger, h.Service.GetAll())
}

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var newPost post.Post
	if err := json.NewDecoder(r.Body).Decode(&newPost); err != nil {
		h.Logger.Error("invalid json", "error", err)
		writeError(w, http.StatusBadRequest, typeError, "invalid JSON payload")
		return
	}

	var claims claims.Claims
	if ok := getClaimsFromContext(w, r, &claims); !ok {
		return
	}

	if err := h.Service.CreatePost(&newPost, claims.User.Username, claims.User.ID); err != nil {
		writeError(w, http.StatusBadRequest, typeError, err.Error())
		return
	}

	if ok := writeJSON(w, h.Logger, newPost); ok {
		h.Logger.Info("new post created", "user", claims.User.ID)
	}
}

func (h *PostHandler) GetPostByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	postID, ok := vars[muxVarPostID]
	if !ok || len(postID) != lenID {
		writeError(w, http.StatusBadRequest, typeMessage, "invalid post id")
		return
	}

	post, err := h.Service.GetByID(postID)
	if err != nil {
		writeError(w, http.StatusNotFound, typeMessage, err.Error())
		return
	}

	writeJSON(w, h.Logger, post)
}

func (h *PostHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	vars := mux.Vars(r)

	postID, ok := vars[muxVarPostID]
	if !ok || len(postID) != lenID {
		writeError(w, http.StatusBadRequest, typeMessage, "invalid post id")
		return
	}

	var comment = make(map[string]string)
	if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
		h.Logger.Error("Invalid JSON", "error", err)
		writeError(w, http.StatusBadRequest, typeError, "invalid JSON payload")
		return
	}

	var claims claims.Claims
	if ok := getClaimsFromContext(w, r, &claims); !ok {
		return
	}

	post, err := h.Service.AddComment(postID, comment["comment"], &claims)
	if err != nil {
		h.Logger.Error("AddComment", "error", err)
		writeError(w, http.StatusBadRequest, typeError, err.Error())
		return
	}

	if ok := writeJSON(w, h.Logger, post); ok {
		h.Logger.Info("new comm created", "user", claims.User.ID)
	}
}

func (h *PostHandler) RemoveComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	postID, ok1 := vars[muxVarPostID]
	if !ok1 {
		writeError(w, http.StatusBadRequest, typeMessage, "invalid post id")
		return
	}

	commID, ok2 := vars[muxVarCommID]
	if !ok2 {
		writeError(w, http.StatusBadRequest, typeMessage, "invalid comment id")
		return
	}

	post, err := h.Service.RemoveComment(postID, commID)
	if err != nil {
		writeError(w, http.StatusNotFound, typeError, err.Error())
		return
	}

	if ok := writeJSON(w, h.Logger, post); ok {
		h.Logger.Info("comment delete", muxVarPostID, postID, muxVarCommID, commID)
	}
}

func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	postID, ok := vars[muxVarPostID]
	if !ok {
		writeError(w, http.StatusBadRequest, typeMessage, "invalid post id")
		return
	}

	if err := h.Service.Delete(postID); err != nil {
		writeError(w, http.StatusNotFound, typeError, err.Error())
		return
	}

	if ok := writeJSON(w, h.Logger, map[string]string{"message": "success"}); ok {
		h.Logger.Info("post delete", muxVarPostID, postID)
	}
}

func (h *PostHandler) AddVote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	postID, ok1 := vars[muxVarPostID]
	if !ok1 {
		writeError(w, http.StatusBadRequest, typeMessage, "invalid post id")
		return
	}

	action, ok2 := vars[muxVarAction]
	if !ok2 {
		writeError(w, http.StatusBadRequest, typeMessage, "invalid vote action")
		return
	}

	var claims claims.Claims
	if ok := getClaimsFromContext(w, r, &claims); !ok {
		return
	}

	post, err := h.Service.AddVote(postID, claims.User.ID, action)
	if err != nil {
		writeError(w, http.StatusBadRequest, typeError, err.Error())
		return
	}

	if ok := writeJSON(w, h.Logger, post); ok {
		h.Logger.Info("user voting", "user", claims.User.ID, muxVarAction, action)
	}
}

func (h *PostHandler) GetPostsByUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	userID, ok := vars[muxVarLogin]
	if !ok {
		writeError(w, http.StatusBadRequest, typeMessage, "invalid user id")
		return
	}

	posts := h.Service.GetByUser(userID)

	writeJSON(w, h.Logger, posts)
}

func (h *PostHandler) GetPostsByCategory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	category, ok := vars[muxVarCategory]
	if !ok {
		writeError(w, http.StatusBadRequest, typeMessage, "invalid category")
		return
	}

	posts := h.Service.GetByCategory(category)

	writeJSON(w, h.Logger, posts)
}

func writeJSON(w http.ResponseWriter, logger *slog.Logger, data any) bool {
	resp, err := json.Marshal(data)
	if err != nil {
		logger.Error("Failed to serialize JSON response", "error", err)
		writeError(w, http.StatusInternalServerError, typeError, "failed json marshal")
		return false
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write(resp); err != nil {
		logger.Error("Failed to write response to client", "error", err)
		return false
	}
	return true
}

func getClaimsFromContext(w http.ResponseWriter, r *http.Request, c *claims.Claims) bool {
	val, ok := r.Context().Value(claims.TokenContextKey).(*claims.Claims)
	if !ok || val == nil || val.User.ID == "" {
		writeError(w, http.StatusUnauthorized, typeMessage, "unauthorized")
		return false
	}
	*c = *val
	return true
}

func writeError(w http.ResponseWriter, status int, field, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{field: msg}); err != nil {
		return
	}
}
