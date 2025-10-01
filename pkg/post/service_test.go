package post_test

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"redditclone/pkg/claims"
	"redditclone/pkg/post"
	"redditclone/pkg/post/mocks"
	"redditclone/pkg/user"
)

func resetMock(m *mocks.RepoPost) {
	m.ExpectedCalls = nil
	m.Calls = nil
}

var (
	mockRepo *mocks.RepoPost
	service  *post.PostService
	expected *post.Post

	defaultClaims = &claims.Claims{
		User: struct {
			Username string `json:"username"`
			ID       string `json:"id"`
		}{
			Username: "testuser",
			ID:       "user123",
		},
	}
)

func TestMain(m *testing.M) {
	expected = &post.Post{Title: "Testing"}
	mockRepo = new(mocks.RepoPost)
	service = post.NewService(mockRepo)

	code := m.Run()
	os.Exit(code)
}

func TestCreatePost(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		defer resetMock(mockRepo)

		p := &post.Post{Title: "Test"}
		mockRepo.On("Create", mock.AnythingOfType("*post.Post")).Return(nil)

		err := service.CreatePost(p, "user", "id")

		assert.NoError(t, err)
		assert.Equal(t, 1, p.Score)
		assert.Equal(t, 0, p.Views)
		assert.Equal(t, 100, p.UpvotePercentage)
		assert.Equal(t, "user", p.Author.Username)
		assert.Equal(t, "id", p.Author.ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("mongo request error", func(t *testing.T) {
		defer resetMock(mockRepo)

		p := &post.Post{Title: "Test"}
		mockRepo.On("Create", mock.AnythingOfType("*post.Post")).Return(errors.New("mongo_err"))

		err := service.CreatePost(p, "user", "id")

		assert.Error(t, err)
		assert.Equal(t, "user", p.Author.Username)
		assert.Equal(t, "id", p.Author.ID)
		mockRepo.AssertExpectations(t)
	})

}

func TestGetAll(t *testing.T) {
	defer resetMock(mockRepo)

	mockPosts := []*post.Post{{Title: "A"}, {Title: "B"}}
	mockRepo.On("GetAll").Return(mockPosts)

	res := service.GetAll()

	assert.Equal(t, 2, len(res))
	mockRepo.AssertExpectations(t)
}

func TestGetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		defer resetMock(mockRepo)

		mockRepo.On("GetByID", "123").Return(expected, nil)

		res, err := service.GetByID("123")

		assert.NoError(t, err)
		assert.Equal(t, expected, res)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetById fail", func(t *testing.T) {
		defer resetMock(mockRepo)

		mockRepo.On("GetByID", "123").Return(nil, errors.New("mongo error"))

		res, err := service.GetByID("123")

		assert.Error(t, err)
		assert.Nil(t, res)
		mockRepo.AssertExpectations(t)
	})

}

func TestAddComment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		defer resetMock(mockRepo)

		mockRepo.On("AddComment", "123", mock.AnythingOfType("post.Comment")).Return(expected, nil)

		res, err := service.AddComment("123", "Nice post!", defaultClaims)

		assert.NoError(t, err)
		assert.Equal(t, expected, res)
		mockRepo.AssertExpectations(t)
	})

	t.Run("AddComment fail", func(t *testing.T) {
		defer resetMock(mockRepo)

		mockRepo.On("AddComment", "123", mock.AnythingOfType("post.Comment")).Return(nil, errors.New("mongo error"))

		res, err := service.AddComment("123", "Nice post!", defaultClaims)

		assert.Error(t, err)
		assert.Nil(t, res)
		mockRepo.AssertExpectations(t)
	})
}

func TestRemoveComment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		defer resetMock(mockRepo)

		mockRepo.On("RemoveComment", "123", "c1").Return(expected, nil)

		res, err := service.RemoveComment("123", "c1")

		assert.NoError(t, err)
		assert.Equal(t, expected, res)
		mockRepo.AssertExpectations(t)
	})

	t.Run("remove comment fail", func(t *testing.T) {
		defer resetMock(mockRepo)

		mockRepo.On("RemoveComment", "123", "c1").Return(nil, errors.New("mongo error"))

		res, err := service.RemoveComment("123", "c1")

		assert.Error(t, err)
		assert.Nil(t, res)
		mockRepo.AssertExpectations(t)
	})

}

func TestDelete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		defer resetMock(mockRepo)

		mockRepo.On("Delete", "123").Return(nil)

		err := service.Delete("123")

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("delete fail", func(t *testing.T) {
		defer resetMock(mockRepo)

		mockRepo.On("Delete", "123").Return(errors.New("mongo error"))

		err := service.Delete("123")

		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

}

func TestAddVote(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		defer resetMock(mockRepo)

		mockRepo.On("AddVote", "123", post.Voting{User: "u", Vote: 1}).Return(expected, nil)

		res, err := service.AddVote("123", "u", "upvote")

		assert.NoError(t, err)
		assert.Equal(t, expected, res)
		mockRepo.AssertExpectations(t)
	})

	t.Run("cancel/error", func(t *testing.T) {
		defer resetMock(mockRepo)

		mockRepo.On("CancelVote", "123", "u").Return(expected, nil)
		mockRepo.On("CancelVote", "123", "oops").Return(nil, errors.New("invalid action"))

		res, err := service.AddVote("123", "u", "unvote")

		assert.NoError(t, err)
		assert.Equal(t, expected, res)

		res, err = service.AddVote("123", "oops", "unvote")

		assert.Error(t, err)
		assert.Nil(t, res)

		res, err = service.AddVote("123", "user", "bad_action")

		assert.Error(t, err)
		assert.Equal(t, "invalid action", err.Error())
		assert.Nil(t, res)

		mockRepo.AssertExpectations(t)
	})

}

func TestGetByUser(t *testing.T) {
	defer resetMock(mockRepo)

	posts := []*post.Post{{Author: user.User{Username: "u"}}}
	mockRepo.On("GetByUser", "u").Return(posts)

	res := service.GetByUser("u")

	assert.Equal(t, posts, res)
	mockRepo.AssertExpectations(t)
}

func TestGetByCategory(t *testing.T) {
	defer resetMock(mockRepo)
	posts := []*post.Post{{Category: "tech"}}
	mockRepo.On("GetByCategory", "tech").Return(posts)

	res := service.GetByCategory("tech")

	assert.Equal(t, posts, res)
	mockRepo.AssertExpectations(t)
}
