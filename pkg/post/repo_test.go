package post_test

import (
	"encoding/binary"
	"testing"

	"redditclone/pkg/post"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestGetAllRepo(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success with non valid json", func(mt *mtest.T) {
		posts := []bson.D{
			{{Key: "_id", Value: primitive.NewObjectID()}, {Key: "score", Value: 10}},
			{{Key: "_id", Value: primitive.NewObjectID()}, {Key: "score", Value: 20}},
			{{Key: "_id", Value: "oops"}, {Key: "score", Value: 20}},
		}
		mt.AddMockResponses(mtest.CreateCursorResponse(1, "posts.foo", mtest.FirstBatch, posts...))
		repo := post.NewMongoRepo(mt.DB)

		results := repo.GetAll()

		assert.Len(t, results, 2)
		assert.GreaterOrEqual(t, results[0].Score, results[1].Score)
	})

	mt.Run("mongo Find error", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)

		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{
			Code:    123,
			Message: "some error",
		}))

		results := repo.GetAll()

		assert.Nil(t, results)
	})
}

func TestGetByUserRepo(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		user := "alice"
		posts := []bson.D{
			{
				{Key: "_id", Value: primitive.NewObjectID()},
				{Key: "author", Value: bson.M{"username": user}},
			},
		}
		mt.AddMockResponses(mtest.CreateCursorResponse(1, "posts.foo", mtest.FirstBatch, posts...))

		repo := post.NewMongoRepo(mt.DB)
		results := repo.GetByUser(user)

		assert.Len(t, results, 1)
		assert.Equal(t, user, results[0].Author.Username)
	})

	mt.Run("post not found", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)
		validID := "60b6d28f3f1d2f8a2c0d6b5a"

		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{
			Message: "error",
		}))

		result, err := repo.GetByID(validID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.EqualError(t, err, "failed to increment views and fetch post: error")
	})
}

func TestGetByCategoryRepo(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		category := "tech"
		posts := []bson.D{
			{
				{Key: "_id", Value: primitive.NewObjectID()},
				{Key: "category", Value: category},
			},
		}
		mt.AddMockResponses(mtest.CreateCursorResponse(1, "posts.foo", mtest.FirstBatch, posts...))

		repo := post.NewMongoRepo(mt.DB)
		results := repo.GetByCategory(category)

		assert.Len(t, results, 1)
		assert.Equal(t, category, results[0].Category)
	})
}

func TestDeleteRepo(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("invalid ID format", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)
		err := repo.Delete("invalid")
		assert.EqualError(t, err, "invalid ID format")
	})

	mt.Run("delete success", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse(
			bson.E{Key: "n", Value: 1},
			bson.E{Key: "ok", Value: 1},
		))
		repo := post.NewMongoRepo(mt.DB)
		err := repo.Delete(primitive.NewObjectID().Hex())
		assert.NoError(t, err)
	})

	mt.Run("delete error", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)

		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{
			Code:    123,
			Message: "simulated delete error",
		}))

		err := repo.Delete(primitive.NewObjectID().Hex())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "simulated delete error")
	})

	mt.Run("post not found", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)

		mt.AddMockResponses(mtest.CreateSuccessResponse(
			bson.E{Key: "ok", Value: 1},
			bson.E{Key: "n", Value: 0},
		))

		err := repo.Delete(primitive.NewObjectID().Hex())

		assert.EqualError(t, err, "post not found")
	})
}

func TestMongoRepo_AddComment(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)

		mongoID := primitive.NewObjectID()
		hexMongoID := mongoID.Hex()

		commentArg := post.Comment{
			Body: "lyalyalya",
		}

		update := bson.D{
			{Key: "_id", Value: mongoID},
			{Key: "comments", Value: bson.A{
				bson.D{
					{Key: "body", Value: "lyalyalya"},
				},
			}},
		}

		mt.AddMockResponses(
			bson.D{
				{Key: "ok", Value: 1},
				{Key: "value", Value: update},
			},
		)

		resp, err := repo.AddComment(hexMongoID, commentArg)

		assert.NoError(t, err)
		assert.Equal(t, 1, len(resp.Comments))
		assert.Equal(t, "lyalyalya", resp.Comments[0].Body)
	})

	mt.Run("bad post id", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)
		_, err := repo.AddComment("🦧", post.Comment{})

		assert.Error(t, err)
	})

	mt.Run("err no document", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)

		mt.AddMockResponses(
			bson.D{
				{Key: "ok", Value: 1},
				{Key: "value", Value: nil},
			},
		)

		_, err := repo.AddComment("507f1f77bcf86cd799439011", post.Comment{
			Body: "test comment",
		})

		assert.Error(t, err)
		assert.Equal(t, "post not found", err.Error())
	})

	mt.Run("unexpected mongo error", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)

		mt.AddMockResponses(mtest.CreateCommandErrorResponse(
			mtest.CommandError{
				Code:    91,
				Message: "server is shutting down",
				Name:    "ShutdownInProgress",
			},
		))

		_, err := repo.AddComment("507f1f77bcf86cd799439011", post.Comment{
			Body: "test comment",
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add comment")
	})
}

func TestRemoveCommentRepo(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)

		mongoID := primitive.NewObjectID()
		hexMongoID := mongoID.Hex()

		update := bson.D{
			{Key: "_id", Value: mongoID},
			{Key: "comments", Value: bson.A{}},
		}

		mt.AddMockResponses(
			bson.D{
				{Key: "ok", Value: 1},
				{Key: "value", Value: update},
			},
		)

		resp, err := repo.RemoveComment(hexMongoID, "123456789012345678901234")

		assert.NoError(t, err)
		assert.Equal(t, 0, len(resp.Comments))
	})

	mt.Run("bad post id", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)
		_, err := repo.RemoveComment("🦧", "ugabuga")

		assert.Error(t, err)
	})

	mt.Run("err no document", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)

		mt.AddMockResponses(
			bson.D{
				{Key: "ok", Value: 1},
				{Key: "value", Value: nil},
			},
		)

		_, err := repo.RemoveComment("507f1f77bcf86cd799439011", "🦧")

		assert.Error(t, err)
		assert.Equal(t, "post not found", err.Error())
	})

	mt.Run("unexpected mongo error", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)

		mt.AddMockResponses(mtest.CreateCommandErrorResponse(
			mtest.CommandError{
				Code:    91,
				Message: "server is shutting down",
				Name:    "ShutdownInProgress",
			},
		))

		_, err := repo.RemoveComment("507f1f77bcf86cd799439011", "🦧")

		assert.Error(t, err)
	})
}

func TestAddVoteRepo(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	vote := post.Voting{
		User: "test_user",
		Vote: 1,
	}

	mt.Run("success", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)
		mongoID := primitive.NewObjectID()
		hexMongoID := mongoID.Hex()

		update := bson.D{
			{Key: "_id", Value: mongoID},
			{Key: "comments", Value: bson.A{
				bson.D{
					{Key: "body", Value: "lyalyalya"},
				},
			}},
			{Key: "votes", Value: bson.A{
				bson.D{
					{Key: "user", Value: "ugabuga"},
					{Key: "vote", Value: 1},
				},
				bson.D{
					{Key: "user", Value: "test_user"},
					{Key: "vote", Value: -1},
				},
			}},
		}

		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, update),
			mtest.CreateCursorResponse(0, "foo.bar", mtest.NextBatch),
			bson.D{ // ReplaceOne
				{Key: "ok", Value: 1},
				{Key: "n", Value: 1},
				{Key: "nModified", Value: 1},
			},
		)

		_, err := repo.AddVote(hexMongoID, vote)

		assert.NoError(t, err)
	})

	mt.Run("bad id", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)
		_, err := repo.AddVote("🦧", vote)

		assert.Error(t, err)

	})

	mt.Run("success", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)
		mongoID := primitive.NewObjectID()
		hexMongoID := mongoID.Hex()

		update := bson.D{
			{Key: "_id", Value: mongoID},
			{Key: "comments", Value: bson.A{
				bson.D{
					{Key: "body", Value: "lyalyalya"},
				},
			}},
			{Key: "votes", Value: bson.A{}},
		}

		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, update),
			mtest.CreateCursorResponse(0, "foo.bar", mtest.NextBatch),
			bson.D{
				{Key: "ok", Value: 1},
				{Key: "n", Value: 1},
				{Key: "nModified", Value: 1},
			},
		)

		_, err := repo.AddVote(hexMongoID, vote)

		assert.NoError(t, err)
	})
}

func TestCancelVoteRepo(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)
		mongoID := primitive.NewObjectID()
		hexMongoID := mongoID.Hex()

		update := bson.D{
			{Key: "_id", Value: mongoID},
			{Key: "comments", Value: bson.A{
				bson.D{
					{Key: "body", Value: "lyalyalya"},
				},
			}},
			{Key: "votes", Value: bson.A{
				bson.D{
					{Key: "user", Value: "test_user"},
					{Key: "vote", Value: 1},
				},
			}},
		}

		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, update),
			mtest.CreateCursorResponse(0, "foo.bar", mtest.NextBatch),
			bson.D{
				{Key: "ok", Value: 1},
				{Key: "n", Value: 1},
				{Key: "nModified", Value: 1},
			},
		)

		_, err := repo.CancelVote(hexMongoID, "test_user")

		assert.NoError(t, err)
	})

	mt.Run("success", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)
		_, err := repo.CancelVote("🦧", "test_user")

		assert.Error(t, err)
	})

	mt.Run("success", func(mt *mtest.T) {

		repo := post.NewMongoRepo(mt.DB)
		mongoID := primitive.NewObjectID()
		hexMongoID := mongoID.Hex()

		update := bson.D{
			{Key: "_id", Value: mongoID},
			{Key: "comments", Value: bson.A{
				bson.D{
					{Key: "body", Value: "lyalyalya"},
				},
			}},
			{Key: "votes", Value: bson.A{}},
		}

		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, update),
			mtest.CreateCursorResponse(0, "foo.bar", mtest.NextBatch),
			bson.D{
				{Key: "ok", Value: 1},
				{Key: "n", Value: 1},
				{Key: "nModified", Value: 1},
			},
		)

		_, err := repo.CancelVote(hexMongoID, "test_user")

		assert.Error(t, err)
	})

}

func TestMongoRepo_Create(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("successfully insert post", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)

		var p post.Post
		expectedID := primitive.NewObjectID()

		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "insertedId", Value: expectedID},
		})

		err := repo.Create(&p)

		// это проверка на то, что дейтсвитедбно создался новый пост с новым внутренним монгоID
		// три последних байта отвечают за инкремент, после каждого инсерта он должен быть на 1 больше
		lastThreeBytes := expectedID[9:12]

		// берем три последних байта в число и прибавляем единицу
		counter := binary.BigEndian.Uint32(append([]byte{0}, lastThreeBytes...))
		counter++

		// это для кольца, чтобы не переполнилось
		newBytes := counter & 0xFFFFFF
		// преобразуем число в слайс байтов сдвигами
		resultBytes := []byte{
			byte(newBytes >> 16),
			byte(newBytes >> 8),
			byte(newBytes),
		}

		// соединяем три этих байта с прежними 9 байтами
		res := make([]byte, 12)
		copy(res, expectedID[:9])
		copy(res[9:], resultBytes)

		// берем текуший id, который вернул вызов функции
		actual := p.MongoID[:]

		assert.NoError(t, err)
		// и сравниваем их
		assert.Equal(t, res, actual)
	})

	mt.Run("error insert one", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)

		var p post.Post

		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: nil},
			{Key: "insertedId", Value: nil},
		})

		err := repo.Create(&p)

		assert.Error(t, err)
	})

	mt.Run("error insert one", func(mt *mtest.T) {
		repo := post.NewMongoRepo(mt.DB)

		p := &post.Post{
			ID: "abc123",
		}

		mt.AddMockResponses(
			mtest.CreateWriteErrorsResponse(
				mtest.WriteError{
					Index:   0,
					Code:    11000,
					Message: "E11000 duplicate key error collection: test.posts index: id dup key",
				},
			),
		)

		err := repo.Create(p)

		assert.Error(t, err)
		assert.EqualError(t, err, "post already exists")
	})

}
