package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Folder struct {
	ID        primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Name      string              `bson:"name" json:"name"`
	OwnerID   primitive.ObjectID  `bson:"ownerId" json:"ownerId"`
	ParentID  *primitive.ObjectID `bson:"parentId" json:parentId,omitempty`
	CreatedAt time.Time           `bson:"createdAt" json:createdAt`
	UpdatedAt time.Time           `bson:"updatedAt" json:updatedAt`
}
