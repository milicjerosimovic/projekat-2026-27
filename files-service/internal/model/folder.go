package model

import (
    "time"

    "go.mongodb.org/mongo-driver/bson/primitive"
)

type Folder struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Name      string  `bson:"name"`
	OwnerID   primitive.ObjectID `bson:"ownerId"`
	ParentID  *primitive.ObjectID `bson:"parentId"`
	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"upadetedAt"`
}
