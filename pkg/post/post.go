package post

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"redditclone/pkg/user"
)

type Comment struct {
	Created time.Time `json:"created" bson:"created"`
	Author  user.User `json:"author" bson:"author"`
	Body    string    `json:"body" bson:"body"`
	ID      string    `json:"id" bson:"id"`
}

type Voting struct {
	User string `json:"user"`
	Vote int8   `json:"vote"`
}

type Post struct {
	MongoID          primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	Score            int                `json:"score"`
	Views            int                `json:"views"`
	Type             string             `json:"type"`
	Title            string             `json:"title"`
	Author           user.User          `json:"author"`
	Category         string             `json:"category"`
	Text             string             `json:"text,omitempty" bson:"text,omitempty"`
	Votes            []Voting           `json:"votes"`
	Comments         []Comment          `json:"comments"`
	Created          time.Time          `json:"created"`
	UpvotePercentage int                `json:"upvotePercentage"`
	ID               string             `json:"id" bson:"-"`
	URL              *string            `json:"url,omitempty" bson:"url,omitempty"`
}

type Repository interface {
	Create(post *Post) error
	GetByID(id string) (*Post, error)
	GetAll() []*Post
	GetByUser(userID string) []*Post
	GetByCategory(category string) []*Post
	Delete(postID string) error
	AddComment(postID string, comment Comment) (*Post, error)
	RemoveComment(postID string, commentID string) (*Post, error)
	AddVote(postID string, vote Voting) (*Post, error)
	CancelVote(postID string, user string) (*Post, error)
}
