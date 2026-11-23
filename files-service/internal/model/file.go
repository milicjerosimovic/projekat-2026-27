package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type File struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OwnerID    primitive.ObjectID `bson:"ownerId" json:"ownerId"`
	ParentID   primitive.ObjectID `bson:"parentId" json:"parentId"`
	Name       string             `bson:"name" json:"name"`
	Size       int64              `bson:"size" json:"size"`
	Type       string             `bson:"type" json:"type"`
	CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time          `bson:"updatedAt" json:"updatedAt"`
	ContentRef string             `bson:"contentRef" json:"contentRef"`
}
