package post

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepo struct {
	collection *mongo.Collection
}

func NewMongoRepo(db *mongo.Database) *MongoRepo {
	return &MongoRepo{
		collection: db.Collection("posts"),
	}
}

func (r *MongoRepo) Create(post *Post) error {
	ctx := context.TODO()

	result, err := r.collection.InsertOne(ctx, post)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("post already exists")
		}
		return err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		post.MongoID = oid
		post.ID = oid.Hex()
	} else {
		return errors.New("failed to convert inserted ID to ObjectID")
	}

	return nil
}

func (r *MongoRepo) GetByID(id string) (*Post, error) {
	ctx := context.TODO()
	var post Post

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid ID format")
	}

	err = r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$inc": bson.M{"views": 1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&post)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, errors.New("post not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to increment views and fetch post: %w", err)
	}

	post.ID = post.MongoID.Hex()
	return &post, nil
}

func (r *MongoRepo) GetAll() []*Post {
	ctx := context.TODO()
	cursor, err := r.collection.Find(ctx, bson.D{})
	if err != nil {
		return nil
	}
	defer cursor.Close(ctx)

	var posts []*Post
	for cursor.Next(ctx) {
		var post Post
		if err := cursor.Decode(&post); err != nil {
			continue
		}
		post.ID = post.MongoID.Hex()
		posts = append(posts, &post)
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Score > posts[j].Score
	})

	return posts
}

func (r *MongoRepo) GetByUser(username string) []*Post {
	ctx := context.TODO()
	cursor, err := r.collection.Find(ctx, bson.M{"author.username": username})
	if err != nil {
		return nil
	}
	defer cursor.Close(ctx)

	var posts []*Post
	for cursor.Next(ctx) {
		var post Post
		if cursor.Decode(&post) == nil {
			post.ID = post.MongoID.Hex()
			posts = append(posts, &post)
		}
	}
	return posts
}

func (r *MongoRepo) GetByCategory(category string) []*Post {
	ctx := context.TODO()
	cursor, err := r.collection.Find(ctx, bson.M{"category": category})
	if err != nil {
		return nil
	}
	defer cursor.Close(ctx)

	var posts []*Post
	for cursor.Next(ctx) {
		var post Post
		if cursor.Decode(&post) == nil {
			post.ID = post.MongoID.Hex()
			posts = append(posts, &post)
		}
	}

	return posts
}

func (r *MongoRepo) Delete(postID string) error {
	ctx := context.TODO()

	objectID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return errors.New("invalid ID format")
	}

	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("post not found")
	}

	return nil
}

func (r *MongoRepo) AddComment(postID string, comment Comment) (*Post, error) {
	ctx := context.TODO()

	objectID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.New("invalid ID format")
	}

	comment.ID = primitive.NewObjectID().Hex()

	update := bson.M{
		"$push": bson.M{
			"comments": comment,
		},
	}

	var updatedPost Post
	err = r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updatedPost)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("post not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}

	updatedPost.ID = updatedPost.MongoID.Hex()

	log.Println("updated post with comment:", updatedPost)

	return &updatedPost, nil
}

func (r *MongoRepo) RemoveComment(postID, commentID string) (*Post, error) {
	ctx := context.TODO()

	objectID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.New("invalid post ID format")
	}

	update := bson.M{
		"$pull": bson.M{
			"comments": bson.M{"id": commentID},
		},
	}

	var updatedPost Post
	err = r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updatedPost)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("post not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to remove comment: %w", err)
	}

	updatedPost.ID = updatedPost.MongoID.Hex()
	return &updatedPost, nil
}

func (r *MongoRepo) AddVote(postID string, vote Voting) (*Post, error) {
	ctx := context.TODO()

	post, err := r.FindByID(postID)
	if err != nil {
		return nil, err
	}

	override := false
	for i, v := range post.Votes {
		if v.User == vote.User {
			diff := int(vote.Vote - v.Vote)
			post.Votes[i].Vote = vote.Vote
			post.Score += diff
			override = true
			break
		}
	}

	if !override {
		post.Votes = append(post.Votes, vote)
		post.Score += int(vote.Vote)
	}

	r.updateUpvotePercentage(post)

	objectID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, fmt.Errorf("failed attempt to generate a mongo id")
	}

	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, post)
	return post, err
}

func (r *MongoRepo) CancelVote(postID string, user string) (*Post, error) {
	ctx := context.TODO()
	post, err := r.FindByID(postID)
	if err != nil {
		return nil, err
	}

	for i, v := range post.Votes {
		if v.User == user {
			post.Score -= int(v.Vote)
			post.Votes = append(post.Votes[:i], post.Votes[i+1:]...)

			r.updateUpvotePercentage(post)

			objectID, err := primitive.ObjectIDFromHex(postID)
			if err != nil {
				return nil, fmt.Errorf("failed attempt to generate a mongo id")
			}

			_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, post)
			return post, err
		}
	}

	return nil, errors.New("vote not found")
}

func (r *MongoRepo) updateUpvotePercentage(post *Post) {
	total := len(post.Votes)
	if total == 0 {
		post.UpvotePercentage = 0
		return
	}

	upvotes := 0
	for _, v := range post.Votes {
		if v.Vote == 1 {
			upvotes++
		}
	}
	post.UpvotePercentage = (upvotes * 100) / total
}

func (r *MongoRepo) FindByID(id string) (*Post, error) {
	ctx := context.TODO()
	var post Post

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid ID format")
	}

	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&post)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, errors.New("post not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch post: %w", err)
	}

	post.ID = post.MongoID.Hex()
	return &post, nil
}
