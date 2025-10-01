package post

import (
	"errors"
	"time"

	"redditclone/pkg/claims"
	"redditclone/pkg/user"
)

type ServicePost interface {
	GetAll() []*Post
	CreatePost(post *Post, username, id string) error
	GetByID(id string) (*Post, error)
	AddComment(postID, comment string, claims *claims.Claims) (*Post, error)
	RemoveComment(postID, commID string) (*Post, error)
	Delete(postID string) error
	AddVote(postID, username, action string) (*Post, error)
	GetByUser(username string) []*Post
	GetByCategory(category string) []*Post
}

type PostService struct {
	Repo Repository
}

func NewService(repo Repository) *PostService {
	return &PostService{Repo: repo}
}

func (s *PostService) GetAll() []*Post {
	return s.Repo.GetAll()
}

func (s *PostService) CreatePost(post *Post, username, id string) error {
	post.Score = 1
	post.Views = 0
	post.Author = user.User{
		Username: username,
		ID:       id,
	}
	post.Votes = []Voting{{User: id, Vote: 1}}
	post.Created = time.Now()
	post.UpvotePercentage = 100
	/* без этого монга будет выдавать ошибку, что
	поле comments пустое (nil) и будет невозможно оставить первый
	комменатрий под постом, post.Comments = []Comments{} не работает */
	post.Comments = make([]Comment, 0, 1)

	return s.Repo.Create(post)
}

func (s *PostService) GetByID(id string) (*Post, error) {
	return s.Repo.GetByID(id)
}

func (s *PostService) AddComment(postID, comment string, claims *claims.Claims) (*Post, error) {
	ReadyComment := Comment{
		Created: time.Now(),
		Author: user.User{
			Username: claims.User.Username,
			ID:       claims.User.ID,
		},
		Body: comment,
	}

	return s.Repo.AddComment(postID, ReadyComment)
}

func (s *PostService) RemoveComment(postID, commID string) (*Post, error) {
	return s.Repo.RemoveComment(postID, commID)
}

func (s *PostService) Delete(postID string) error {
	return s.Repo.Delete(postID)
}

func (s *PostService) AddVote(postID, username, action string) (post *Post, err error) {
	if username == "" {
		return nil, errors.New("missing username")
	}

	switch action {
	case "upvote":
		post, err = s.Repo.AddVote(postID, Voting{User: username, Vote: 1})
	case "downvote":
		post, err = s.Repo.AddVote(postID, Voting{User: username, Vote: -1})
	case "unvote":
		post, err = s.Repo.CancelVote(postID, username)
	default:
		return nil, errors.New("invalid action")
	}

	return post, err
}

func (s *PostService) GetByUser(username string) []*Post {
	return s.Repo.GetByUser(username)
}

func (s *PostService) GetByCategory(category string) []*Post {
	return s.Repo.GetByCategory(category)
}
