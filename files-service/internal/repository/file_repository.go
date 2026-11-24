package repository

import (
	"context"
	"time"

	"folder.go/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type FileRepository struct {
	collection *mongo.Collection
}

func NewFileRepository(db *mongo.Database) *FileRepository {
	return &FileRepository{
		collection: db.Collection("files"),
	}
}

func (r *FileRepository) CreateFile(file *model.File) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if file.ID.IsZero() {
		file.ID = primitive.NewObjectID()
	}

	now := time.Now().UTC()
	file.CreatedAt = now
	file.UpdatedAt = now

	_, err := r.collection.InsertOne(ctx, file)
	return err
}

func (r *FileRepository) GetFileByID(
	id primitive.ObjectID,
	ownerID primitive.ObjectID,
) (*model.File, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var file model.File

	err := r.collection.FindOne(ctx, bson.M{
		"_id":     id,
		"ownerId": ownerID,
	}).Decode(&file)

	if err != nil {
		return nil, err
	}

	return &file, nil
}

func (r *FileRepository) GetFilesByParentID(
	ownerID primitive.ObjectID,
	parentID primitive.ObjectID,
) ([]model.File, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{
		"ownerId":  ownerID,
		"parentId": parentID,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	files := make([]model.File, 0)

	if err := cursor.All(ctx, &files); err != nil {
		return nil, err
	}

	return files, nil
}

func (r *FileRepository) RenameFile(
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

func (r *FileRepository) UpdateFileVersion(
	id primitive.ObjectID,
	ownerID primitive.ObjectID,
	size int64,
	fileType string,
	contentRef string,
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
				"size":       size,
				"type":       fileType,
				"contentRef": contentRef,
				"updatedAt":  time.Now().UTC(),
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

func (r *FileRepository) GetTotalStorageUsed(
	ownerID primitive.ObjectID,
) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{
		"ownerId": ownerID,
	})
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var files []model.File
	if err := cursor.All(ctx, &files); err != nil {
		return 0, err
	}

	var total int64
	for _, file := range files {
		total += file.Size
	}

	return total, nil
}
