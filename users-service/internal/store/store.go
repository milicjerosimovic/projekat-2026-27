package store

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"users-service/internal/model"
)

var (
	ErrEmailTaken = errors.New("email already registered")
	ErrNotFound   = errors.New("not found")

	ErrPendingExists    = errors.New("pending quota request exists")
	ErrAlreadyProcessed = errors.New("quota request already processed")
)

type Store struct {
	client    *mongo.Client
	users     *mongo.Collection
	tokens    *mongo.Collection
	revoked   *mongo.Collection
	quotaReqs *mongo.Collection
}

func New(ctx context.Context, uri, dbName string, logger *slog.Logger) (*Store, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	var pingErr error
	for attempt := 1; attempt <= 30; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		pingErr = client.Ping(pingCtx, nil)
		cancel()
		if pingErr == nil {
			break
		}
		logger.Warn("cekam MongoDB", "attempt", attempt)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	if pingErr != nil {
		return nil, pingErr
	}

	db := client.Database(dbName)
	s := &Store{
		client:    client,
		users:     db.Collection("users"),
		tokens:    db.Collection("tokens"),
		revoked:   db.Collection("revoked_tokens"),
		quotaReqs: db.Collection("quota_requests"),
	}
	if err := s.ensureIndexes(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) ensureIndexes(ctx context.Context) error {
	if _, err := s.users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	if _, err := s.tokens.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		{Keys: bson.D{{Key: "tokenHash", Value: 1}, {Key: "type", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "type", Value: 1}}},
	}); err != nil {
		return err
	}
	if _, err := s.revoked.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "jti", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	}); err != nil {
		return err
	}
	if _, err := s.quotaReqs.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"status": model.QuotaPending}),
		},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "createdAt", Value: -1}}},
	}); err != nil {
		return err
	}
	return nil
}

func (s *Store) Ping(ctx context.Context) error { return s.client.Ping(ctx, nil) }

func (s *Store) Close(ctx context.Context) error { return s.client.Disconnect(ctx) }

func (s *Store) CreateUser(ctx context.Context, u *model.User) error {
	res, err := s.users.InsertOne(ctx, u)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrEmailTaken
		}
		return err
	}
	if id, ok := res.InsertedID.(primitive.ObjectID); ok {
		u.ID = id
	}
	return nil
}

func (s *Store) UserByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	err := s.users.FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) UserByID(ctx context.Context, id primitive.ObjectID) (*model.User, error) {
	var u model.User
	err := s.users.FindOne(ctx, bson.M{"_id": id}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) ActivateUser(ctx context.Context, id primitive.ObjectID) error {
	_, err := s.users.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"active": true}})
	return err
}

func (s *Store) SetPassword(ctx context.Context, id primitive.ObjectID, hash string, now time.Time) error {
	_, err := s.users.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"passwordHash":      hash,
		"passwordChangedAt": now,
		"tokensValidAfter":  now,
		"failedLogins":      0,
		"lockedUntil":       time.Time{},
	}})
	return err
}

func (s *Store) RecordFailedLogin(ctx context.Context, id primitive.ObjectID, max int, lockFor time.Duration) error {
	var u model.User
	err := s.users.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$inc": bson.M{"failedLogins": 1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&u)
	if err != nil {
		return err
	}
	if u.FailedLogins >= max {
		_, err = s.users.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
			"lockedUntil":  time.Now().UTC().Add(lockFor),
			"failedLogins": 0,
		}})
	}
	return err
}

func (s *Store) ResetFailedLogins(ctx context.Context, id primitive.ObjectID) error {
	_, err := s.users.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"failedLogins": 0,
		"lockedUntil":  time.Time{},
	}})
	return err
}

func (s *Store) SaveToken(ctx context.Context, t *model.OneTimeToken) error {
	if _, err := s.tokens.DeleteMany(ctx, bson.M{"userId": t.UserID, "type": t.Type}); err != nil {
		return err
	}
	_, err := s.tokens.InsertOne(ctx, t)
	return err
}

func (s *Store) TokenByHash(ctx context.Context, typ, hash string) (*model.OneTimeToken, error) {
	var t model.OneTimeToken
	err := s.tokens.FindOne(ctx, bson.M{
		"tokenHash": hash,
		"type":      typ,
		"used":      false,
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) ActiveTokenForUser(ctx context.Context, userID primitive.ObjectID, typ string) (*model.OneTimeToken, error) {
	var t model.OneTimeToken
	err := s.tokens.FindOne(ctx, bson.M{
		"userId":    userID,
		"type":      typ,
		"used":      false,
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}, options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) ConsumeToken(ctx context.Context, id primitive.ObjectID) (bool, error) {
	res, err := s.tokens.UpdateOne(ctx, bson.M{"_id": id, "used": false}, bson.M{"$set": bson.M{"used": true}})
	if err != nil {
		return false, err
	}
	return res.ModifiedCount == 1, nil
}

func (s *Store) IncrementTokenAttempts(ctx context.Context, id primitive.ObjectID) (int, error) {
	var t model.OneTimeToken
	err := s.tokens.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$inc": bson.M{"attempts": 1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&t)
	if err != nil {
		return 0, err
	}
	return t.Attempts, nil
}

func (s *Store) RevokeJTI(ctx context.Context, jti string, exp time.Time) error {
	_, err := s.revoked.InsertOne(ctx, bson.M{"jti": jti, "expiresAt": exp})
	if err != nil && mongo.IsDuplicateKeyError(err) {
		return nil
	}
	return err
}

func (s *Store) IsRevoked(ctx context.Context, jti string) (bool, error) {
	n, err := s.revoked.CountDocuments(ctx, bson.M{"jti": jti})
	return n > 0, err
}

func (s *Store) UsersPage(ctx context.Context, skip, limit int64) ([]model.User, int64, error) {
	total, err := s.users.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}
	cur, err := s.users.Find(ctx, bson.M{},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}).SetSkip(skip).SetLimit(limit))
	if err != nil {
		return nil, 0, err
	}
	var list []model.User
	if err := cur.All(ctx, &list); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *Store) CreateQuotaRequest(ctx context.Context, q *model.QuotaRequest) error {
	res, err := s.quotaReqs.InsertOne(ctx, q)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrPendingExists
		}
		return err
	}
	if id, ok := res.InsertedID.(primitive.ObjectID); ok {
		q.ID = id
	}
	return nil
}

func (s *Store) QuotaRequests(ctx context.Context, userID primitive.ObjectID, status string) ([]model.QuotaRequest, error) {
	filter := bson.M{}
	if !userID.IsZero() {
		filter["userId"] = userID
	}
	if status != "" {
		filter["status"] = status
	}
	cur, err := s.quotaReqs.Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(200))
	if err != nil {
		return nil, err
	}
	var list []model.QuotaRequest
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (s *Store) ResolveQuotaRequest(ctx context.Context, id primitive.ObjectID, approve bool, adminID primitive.ObjectID) (*model.QuotaRequest, error) {
	status := model.QuotaRejected
	if approve {
		status = model.QuotaApproved
	}

	var q model.QuotaRequest
	err := s.quotaReqs.FindOneAndUpdate(ctx,
		bson.M{"_id": id, "status": model.QuotaPending},
		bson.M{"$set": bson.M{"status": status, "processedAt": time.Now().UTC(), "processedBy": adminID}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&q)
	if errors.Is(err, mongo.ErrNoDocuments) {
		n, countErr := s.quotaReqs.CountDocuments(ctx, bson.M{"_id": id})
		if countErr == nil && n > 0 {
			return nil, ErrAlreadyProcessed
		}
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if approve {
		_, updErr := s.users.UpdateOne(ctx, bson.M{"_id": q.UserID},
			bson.M{"$max": bson.M{"quotaBytes": q.RequestedBytes}})
		if updErr != nil {
			_, _ = s.quotaReqs.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
				"$set":   bson.M{"status": model.QuotaPending},
				"$unset": bson.M{"processedAt": "", "processedBy": ""},
			})
			return nil, updErr
		}
	}
	return &q, nil
}
