package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"redditclone/pkg/claims"
	"redditclone/pkg/handlers"
	"redditclone/pkg/post"
	"redditclone/pkg/post/mocks"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	NicePostID = "123456789012345678901234"
)

var (
	mockPostService *mocks.ServicePost
	handler         *handlers.PostHandler
	logger          *slog.Logger
	defaultComment  = map[string]string{"comment": "test comment"}
	defaultID       = map[string]string{"post_id": NicePostID}
	defaultClaims   = &claims.Claims{
		User: struct {
			Username string `json:"username"`
			ID       string `json:"id"`
		}{
			Username: "testuser",
			ID:       "user123",
		},
	}
)

func resetMock(m *mocks.ServicePost) {
	m.ExpectedCalls = nil
	m.Calls = nil
}

func TestMain(m *testing.M) {
	mockPostService = new(mocks.ServicePost)
	logger = slog.Default()
	handler = handlers.NewPostHandler(mockPostService, logger)

	code := m.Run()
	os.Exit(code)
}

func SetDefaultUserClaims(req *http.Request) *http.Request {
	ctx := context.WithValue(req.Context(), claims.TokenContextKey, defaultClaims)
	return req.WithContext(ctx)
}

func TestCreatePost(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewBufferString(`{"invalid": }`))
		w := httptest.NewRecorder()

		handler.CreatePost(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid JSON payload")
	})

	t.Run("missing claims", func(t *testing.T) {
		body, err := json.Marshal(post.Post{})
		assert.NoError(t, err)

		r := httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.CreatePost(w, r)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "unauthorized")
	})

	t.Run("service error", func(t *testing.T) {
		defer resetMock(mockPostService)

		body, err := json.Marshal(post.Post{})
		assert.NoError(t, err)

		r := SetDefaultUserClaims(httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewReader(body)))
		w := httptest.NewRecorder()

		mockPostService.On("CreatePost", mock.AnythingOfType("*post.Post"), "testuser", "user123").
			Return(errors.New("some_error"))

		handler.CreatePost(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockPostService.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		defer resetMock(mockPostService)

		body, err := json.Marshal(post.Post{})
		assert.NoError(t, err)

		r := SetDefaultUserClaims(httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewReader(body)))
		w := httptest.NewRecorder()

		mockPostService.On("CreatePost", mock.AnythingOfType("*post.Post"), "testuser", "user123").
			Return(nil)

		handler.CreatePost(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		mockPostService.AssertExpectations(t)
	})
}

func TestGetPostByID(t *testing.T) {
	t.Run("invalid id length", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/post/bad_id", nil)
		r = mux.SetURLVars(r, map[string]string{"post_id": "bad_id"})
		w := httptest.NewRecorder()

		handler.GetPostByID(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid post id")
	})

	t.Run("service returns error", func(t *testing.T) {
		defer resetMock(mockPostService)

		r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/post/%s", NicePostID), nil)
		r = mux.SetURLVars(r, map[string]string{"post_id": NicePostID})
		w := httptest.NewRecorder()

		mockPostService.On("GetByID", NicePostID).
			Return(nil, errors.New("not found"))

		handler.GetPostByID(w, r)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "not found")
		mockPostService.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		defer resetMock(mockPostService)

		expected := &post.Post{ID: NicePostID}

		r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/posts/%s", NicePostID), nil)
		r = mux.SetURLVars(r, map[string]string{"post_id": NicePostID})
		w := httptest.NewRecorder()

		mockPostService.On("GetByID", NicePostID).
			Return(expected, nil)

		handler.GetPostByID(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		mockPostService.AssertExpectations(t)
	})
}

func TestAddComment(t *testing.T) {
	t.Run("invalid post ID", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/api/post/id", nil)
		r = mux.SetURLVars(r, map[string]string{"post_id": "short"})
		w := httptest.NewRecorder()

		handler.AddComment(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid post id")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/api/post/id", bytes.NewBufferString("not a json"))
		r = SetDefaultUserClaims(mux.SetURLVars(r, defaultID))
		w := httptest.NewRecorder()

		handler.AddComment(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid JSON payload")
	})

	t.Run("unauthorized (no claims)", func(t *testing.T) {
		body, err := json.Marshal(defaultComment)
		assert.NoError(t, err)

		r := httptest.NewRequest(http.MethodPost, "/api/post/id", bytes.NewBuffer(body))
		r = mux.SetURLVars(r, defaultID)
		w := httptest.NewRecorder()

		handler.AddComment(w, r)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "unauthorized")
	})

	t.Run("success", func(t *testing.T) {
		defer resetMock(mockPostService)

		jsonBody, err := json.Marshal(defaultComment)
		assert.NoError(t, err)

		r := httptest.NewRequest(http.MethodPost, "/api/post/nice_id", bytes.NewBuffer(jsonBody))
		r = SetDefaultUserClaims(mux.SetURLVars(r, defaultID))
		w := httptest.NewRecorder()

		expected := &post.Post{ID: NicePostID}
		mockPostService.On("AddComment", NicePostID, "test comment", defaultClaims).
			Return(expected, nil)

		handler.AddComment(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), NicePostID)
		mockPostService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		defer resetMock(mockPostService)

		jsonBody, err := json.Marshal(defaultComment)
		assert.NoError(t, err)

		r := httptest.NewRequest(http.MethodPost, "/api/post/id", bytes.NewBuffer(jsonBody))
		r = SetDefaultUserClaims(mux.SetURLVars(r, defaultID))
		w := httptest.NewRecorder()

		mockPostService.On("AddComment", NicePostID, "test comment", defaultClaims).
			Return(nil, errors.New("something went wrong"))

		handler.AddComment(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "something went wrong")
		mockPostService.AssertExpectations(t)
	})
}

func TestRemoveComment(t *testing.T) {
	t.Run("missing post id", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodDelete, "/api/post/a_gde/123", nil)
		w := httptest.NewRecorder()

		handler.RemoveComment(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid post id")
	})

	t.Run("missing comment id", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodDelete, "/api/post/nice_id/a_gde", nil)
		r = mux.SetURLVars(r, map[string]string{"post_id": NicePostID})
		w := httptest.NewRecorder()

		handler.RemoveComment(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid comment id")
	})

	t.Run("service error", func(t *testing.T) {
		defer resetMock(mockPostService)

		r := httptest.NewRequest(http.MethodDelete, "/api/post/123/456", nil)
		r = mux.SetURLVars(r, map[string]string{
			"post_id": NicePostID,
			"comm_id": NicePostID,
		})
		w := httptest.NewRecorder()

		mockPostService.On("RemoveComment", NicePostID, NicePostID).
			Return(nil, errors.New("not found"))

		handler.RemoveComment(w, r)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "not found")
		mockPostService.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		defer resetMock(mockPostService)

		expected := &post.Post{ID: NicePostID}

		r := httptest.NewRequest(http.MethodDelete, "/api/post/123/456", nil)
		r = mux.SetURLVars(r, map[string]string{
			"post_id": NicePostID,
			"comm_id": NicePostID,
		})
		w := httptest.NewRecorder()

		mockPostService.On("RemoveComment", NicePostID, NicePostID).
			Return(expected, nil)

		handler.RemoveComment(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), NicePostID)
		mockPostService.AssertExpectations(t)
	})
}

func TestDeletePost(t *testing.T) {
	t.Run("missing post id", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodDelete, "/api/post/a_gde", nil)
		w := httptest.NewRecorder()

		handler.DeletePost(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid post id")
	})

	t.Run("service error", func(t *testing.T) {
		defer resetMock(mockPostService)

		r := httptest.NewRequest(http.MethodDelete, "/api/post/123", nil)
		r = mux.SetURLVars(r, map[string]string{"post_id": NicePostID})
		w := httptest.NewRecorder()

		mockPostService.On("Delete", NicePostID).
			Return(errors.New("post not found"))

		handler.DeletePost(w, r)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "post not found")
		mockPostService.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		defer resetMock(mockPostService)

		r := httptest.NewRequest(http.MethodDelete, "/api/post/123", nil)
		r = mux.SetURLVars(r, map[string]string{"post_id": NicePostID})
		w := httptest.NewRecorder()

		mockPostService.On("Delete", NicePostID).
			Return(nil)

		handler.DeletePost(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "success")
		mockPostService.AssertExpectations(t)
	})
}

func TestAddVote(t *testing.T) {
	t.Run("missing post id", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/api/post//vote/up", nil)
		r = mux.SetURLVars(r, map[string]string{"action": "up"})
		w := httptest.NewRecorder()

		handler.AddVote(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid post id")
	})

	t.Run("missing vote action", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/api/post/123/vote/", nil)
		r = mux.SetURLVars(r, map[string]string{"post_id": "123"})
		w := httptest.NewRecorder()

		handler.AddVote(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid vote action")
	})

	t.Run("unauthorized (no claims)", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/api/post/123/vote/up", nil)
		r = mux.SetURLVars(r, map[string]string{
			"post_id": "123",
			"action":  "upvote",
		})
		w := httptest.NewRecorder()

		handler.AddVote(w, r)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		defer resetMock(mockPostService)

		r := httptest.NewRequest(http.MethodPost, "/api/post/123/vote/down", nil)
		r = SetDefaultUserClaims(mux.SetURLVars(r, map[string]string{
			"post_id": "123",
			"action":  "down",
		}))
		w := httptest.NewRecorder()

		mockPostService.On("AddVote", "123", "user123", "down").
			Return(nil, errors.New("vote failed"))

		handler.AddVote(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "vote failed")
		mockPostService.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		defer resetMock(mockPostService)

		expected := &post.Post{ID: "123"}

		r := httptest.NewRequest(http.MethodPost, "/api/post/123/vote/up", nil)
		r = SetDefaultUserClaims(mux.SetURLVars(r, map[string]string{
			"post_id": "123",
			"action":  "up",
		}))
		w := httptest.NewRecorder()

		mockPostService.On("AddVote", "123", "user123", "up").
			Return(expected, nil)

		handler.AddVote(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"123"`)
		mockPostService.AssertExpectations(t)
	})
}

func TestGetPostsByUser(t *testing.T) {
	t.Run("missing user id", func(t *testing.T) {
		defer resetMock(mockPostService)

		r := httptest.NewRequest(http.MethodGet, "/api/post/user/", nil)
		w := httptest.NewRecorder()

		handler.GetPostsByUser(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid user id")
	})

	t.Run("success", func(t *testing.T) {
		defer resetMock(mockPostService)

		expectedPosts := []*post.Post{
			{ID: "1", Text: "first"},
			{ID: "2", Text: "second"},
		}

		r := httptest.NewRequest(http.MethodGet, "/api/post/user/tester", nil)
		r = mux.SetURLVars(r, map[string]string{"login": "tester"})
		w := httptest.NewRecorder()

		mockPostService.On("GetByUser", "tester").
			Return(expectedPosts)

		handler.GetPostsByUser(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "first")
		assert.Contains(t, w.Body.String(), "second")
		mockPostService.AssertExpectations(t)
	})
}

func TestGetPostsByCategory(t *testing.T) {
	t.Run("missing category", func(t *testing.T) {
		defer resetMock(mockPostService)

		r := httptest.NewRequest(http.MethodGet, "/api/post/category/", nil)
		w := httptest.NewRecorder()

		handler.GetPostsByCategory(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid category")
	})

	t.Run("success", func(t *testing.T) {
		defer resetMock(mockPostService)

		r := httptest.NewRequest(http.MethodGet, "/api/post/music/technology", nil)
		r = mux.SetURLVars(r, map[string]string{"category": "music"})
		w := httptest.NewRecorder()

		expectedPosts := []*post.Post{
			{ID: "1", Text: "tech post 1"},
			{ID: "2", Text: "tech post 2"},
		}

		mockPostService.On("GetByCategory", "music").
			Return(expectedPosts)

		handler.GetPostsByCategory(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "tech post 1")
		assert.Contains(t, w.Body.String(), "tech post 2")
		mockPostService.AssertExpectations(t)
	})
}
