package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	RoleUser  = "RK"
	RoleAdmin = "A"
)

const (
	TokenConfirm = "confirm"
	TokenOTP     = "otp"
	TokenReset   = "reset"
)

type User struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	FirstName         string             `bson:"firstName"`
	LastName          string             `bson:"lastName"`
	Email             string             `bson:"email"`
	PasswordHash      string             `bson:"passwordHash"`
	Role              string             `bson:"role"`
	Active            bool               `bson:"active"`
	QuotaBytes        int64              `bson:"quotaBytes"`
	UsedBytes         int64              `bson:"usedBytes"`
	PasswordChangedAt time.Time          `bson:"passwordChangedAt"`
	TokensValidAfter  time.Time          `bson:"tokensValidAfter"`
	FailedLogins      int                `bson:"failedLogins"`
	LockedUntil       time.Time          `bson:"lockedUntil"`
	CreatedAt         time.Time          `bson:"createdAt"`
}

type OneTimeToken struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    primitive.ObjectID `bson:"userId"`
	Type      string             `bson:"type"`
	TokenHash string             `bson:"tokenHash"`
	Attempts  int                `bson:"attempts"`
	Used      bool               `bson:"used"`
	ExpiresAt time.Time          `bson:"expiresAt"`
	CreatedAt time.Time          `bson:"createdAt"`
}
