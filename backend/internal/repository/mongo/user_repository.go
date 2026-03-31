package mongo

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type userDoc struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty"`
	Username  string                 `bson:"username"`
	Password  string                 `bson:"password"`
	CreatedAt time.Time              `bson:"createdAt"`
	Role      string                 `bson:"role"`
	Status    string                 `bson:"status"`
	MetaInfo  map[string]interface{} `bson:"meta_info,omitempty"`
}

type UserRepository struct {
	collection *mongo.Collection
	timeout    time.Duration
}

func NewUserRepository(db *mongo.Database, timeout time.Duration) *UserRepository {
	return &UserRepository{
		collection: db.Collection("users"),
		timeout:    timeout,
	}
}

func (r *UserRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uq_users_username"),
		},
		{
			Keys:    bson.D{{Key: "role", Value: 1}, {Key: "status", Value: 1}, {Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_users_role_status_created"),
		},
	}
	_, err := r.collection.Indexes().CreateMany(ctx, models)
	return err
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	objectID := primitive.NewObjectID()
	user.ID = objectID.Hex()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}
	if user.Role == "" {
		user.Role = domain.UserRoleUser
	}
	if user.Status == "" {
		user.Status = domain.UserStatusActive
	}

	doc := userDoc{
		ID:        objectID,
		Username:  user.Username,
		Password:  user.Password,
		CreatedAt: user.CreatedAt,
		Role:      string(user.Role),
		Status:    string(user.Status),
		MetaInfo:  user.MetaInfo,
	}

	_, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	var doc userDoc
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &domain.User{
		ID:        doc.ID.Hex(),
		Username:  doc.Username,
		Password:  doc.Password,
		CreatedAt: doc.CreatedAt,
		Role:      normalizeUserRole(doc.Role),
		Status:    normalizeUserStatus(doc.Status),
		MetaInfo:  doc.MetaInfo,
	}, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	var doc userDoc
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &domain.User{
		ID:        doc.ID.Hex(),
		Username:  doc.Username,
		Password:  doc.Password,
		CreatedAt: doc.CreatedAt,
		Role:      normalizeUserRole(doc.Role),
		Status:    normalizeUserStatus(doc.Status),
		MetaInfo:  doc.MetaInfo,
	}, nil
}

func (r *UserRepository) List(ctx context.Context, filter domain.UserListFilter) ([]domain.User, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	query := bson.M{}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.Role != "" {
		query["role"] = filter.Role
	}
	if filter.Keyword != "" {
		query["username"] = bson.M{"$regex": regexp.QuoteMeta(filter.Keyword), "$options": "i"}
	}

	count, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSkip(int64((filter.Page - 1) * filter.Limit)).
		SetLimit(int64(filter.Limit)).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	items := make([]domain.User, 0)
	for cursor.Next(ctx) {
		var doc userDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, 0, err
		}
		items = append(items, domain.User{
			ID:        doc.ID.Hex(),
			Username:  doc.Username,
			CreatedAt: doc.CreatedAt,
			Role:      normalizeUserRole(doc.Role),
			Status:    normalizeUserStatus(doc.Status),
			MetaInfo:  doc.MetaInfo,
		})
	}
	if err := cursor.Err(); err != nil {
		return nil, 0, err
	}

	return items, count, nil
}

func (r *UserRepository) UpdateRole(ctx context.Context, userID string, role domain.UserRole) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return domain.ErrNotFound
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": bson.M{"role": string(role)}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func normalizeUserRole(raw string) domain.UserRole {
	if raw == "" {
		return domain.UserRoleUser
	}
	return domain.UserRole(raw)
}

func normalizeUserStatus(raw string) domain.UserStatus {
	if raw == "" {
		return domain.UserStatusActive
	}
	return domain.UserStatus(raw)
}

func (r *UserRepository) UpdateStatus(ctx context.Context, userID string, status domain.UserStatus) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return domain.ErrNotFound
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": bson.M{"status": string(status)}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID string, password string) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return domain.ErrNotFound
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": bson.M{"password": password}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}
