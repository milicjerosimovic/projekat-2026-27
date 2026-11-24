package repository

import (
	"context"
	"time"

	"folder.go/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type FolderRepository struct {
	collection *mongo.Collection
}

func NewFolderRepository(db *mongo.Database) *FolderRepository {
	return &FolderRepository{
		collection: db.Collection("folders"),
	}
}

func (r *FolderRepository) CreateFolder(folder *model.Folder) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if folder.ID.IsZero() {
		folder.ID = primitive.NewObjectID()
	}

	now := time.Now().UTC()
	folder.CreatedAt = now
	folder.UpdatedAt = now

	_, err := r.collection.InsertOne(ctx, folder)
	return err
}

func (r *FolderRepository) GetFolderByID(
	id primitive.ObjectID,
	ownerID primitive.ObjectID,
) (*model.Folder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var folder model.Folder

	err := r.collection.FindOne(ctx, bson.M{
		"_id":     id,
		"ownerId": ownerID,
	}).Decode(&folder)

	if err != nil {
		return nil, err
	}

	return &folder, nil
}

func (r *FolderRepository) GetFoldersByParentID(
	ownerID primitive.ObjectID,
	parentID *primitive.ObjectID,
) ([]model.Folder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"ownerId": ownerID,
	}

	if parentID == nil {
		filter["$or"] = bson.A{
			bson.M{"parentId": bson.M{"$exists": false}},
			bson.M{"parentId": nil},
		}
	} else {
		filter["parentId"] = *parentID
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	folders := make([]model.Folder, 0)

	if err := cursor.All(ctx, &folders); err != nil {
		return nil, err
	}

	return folders, nil
}

func (r *FolderRepository) RenameFolder(
	id primitive.ObjectID,
	ownerID primitive.ObjectID,
	newName string,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{
			"_id":     id,
			"ownerId": ownerID,
		},
		bson.M{
			"$set": bson.M{
				"name":      newName,
				"updatedAt": time.Now().UTC(),
			},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}
