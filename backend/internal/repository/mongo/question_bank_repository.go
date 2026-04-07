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

type questionBankShareDoc struct {
	UserID     primitive.ObjectID            `bson:"userId"`
	Permission domain.QuestionBankPermission `bson:"permission"`
	GrantedBy  primitive.ObjectID            `bson:"grantedBy"`
	GrantedAt  time.Time                     `bson:"grantedAt"`
	ExpiresAt  *time.Time                    `bson:"expiresAt,omitempty"`
}

type questionBankItemDoc struct {
	QuestionID      primitive.ObjectID  `bson:"questionId"`
	PinnedVersionID *primitive.ObjectID `bson:"pinnedVersionId,omitempty"`
	AddedBy         primitive.ObjectID  `bson:"addedBy"`
	AddedAt         time.Time           `bson:"addedAt"`
	Order           int                 `bson:"order"`
}

type questionBankDoc struct {
	ID          primitive.ObjectID            `bson:"_id,omitempty"`
	Name        string                        `bson:"name"`
	OwnerID     primitive.ObjectID            `bson:"ownerId"`
	Description string                        `bson:"description,omitempty"`
	Visibility  domain.QuestionBankVisibility `bson:"visibility"`
	SharedWith  []questionBankShareDoc        `bson:"sharedWith,omitempty"`
	Items       []questionBankItemDoc         `bson:"items,omitempty"`
	CreatedAt   time.Time                     `bson:"createdAt"`
	UpdatedAt   time.Time                     `bson:"updatedAt"`
}

type QuestionBankRepository struct {
	collection *mongo.Collection
	timeout    time.Duration
}

func NewQuestionBankRepository(db *mongo.Database, timeout time.Duration) *QuestionBankRepository {
	return &QuestionBankRepository{
		collection: db.Collection("question_banks"),
		timeout:    timeout,
	}
}

func (r *QuestionBankRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "ownerId", Value: 1}, {Key: "updatedAt", Value: -1}},
			Options: options.Index().SetName("idx_question_banks_owner_updated"),
		},
		{
			Keys:    bson.D{{Key: "sharedWith.userId", Value: 1}},
			Options: options.Index().SetName("idx_question_banks_shared_user"),
		},
		{
			Keys:    bson.D{{Key: "items.questionId", Value: 1}},
			Options: options.Index().SetName("idx_question_banks_items_qid"),
		},
	}
	_, err := r.collection.Indexes().CreateMany(ctx, models)
	return err
}

func (r *QuestionBankRepository) Create(ctx context.Context, bank *domain.QuestionBank) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	if bank.ID == "" {
		bank.ID = primitive.NewObjectID().Hex()
	}
	now := time.Now().UTC()
	if bank.CreatedAt.IsZero() {
		bank.CreatedAt = now
	}
	bank.UpdatedAt = now

	doc, err := toQuestionBankDoc(*bank)
	if err != nil {
		return err
	}
	_, err = r.collection.InsertOne(ctx, doc)
	return err
}

func (r *QuestionBankRepository) FindByID(ctx context.Context, id string) (*domain.QuestionBank, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	var doc questionBankDoc
	if err := r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	out := toDomainQuestionBank(doc)
	return &out, nil
}

func (r *QuestionBankRepository) FindByIDForUser(ctx context.Context, id, userID string) (*domain.QuestionBank, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	query := bson.M{
		"_id": oid,
		"$or": []bson.M{{"ownerId": uid}, {"sharedWith.userId": uid}},
	}
	var doc questionBankDoc
	if err := r.collection.FindOne(ctx, query).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	out := toDomainQuestionBank(doc)
	return &out, nil
}

func (r *QuestionBankRepository) ListByOwnerOrShared(ctx context.Context, userID string, filter domain.QuestionBankListFilter) ([]domain.QuestionBank, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return []domain.QuestionBank{}, 0, nil
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	query := bson.M{
		"$or": []bson.M{{"ownerId": uid}, {"sharedWith.userId": uid}},
	}
	if filter.Keyword != "" {
		query["name"] = bson.M{"$regex": regexp.QuoteMeta(filter.Keyword), "$options": "i"}
	}

	count, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	cursor, err := r.collection.Find(ctx, query, options.Find().SetSkip(int64((filter.Page-1)*filter.Limit)).SetLimit(int64(filter.Limit)).SetSort(bson.D{{Key: "updatedAt", Value: -1}}))
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	items := make([]domain.QuestionBank, 0)
	for cursor.Next(ctx) {
		var doc questionBankDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, 0, err
		}
		items = append(items, toDomainQuestionBank(doc))
	}
	if err := cursor.Err(); err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

func (r *QuestionBankRepository) UpdateBase(ctx context.Context, bank *domain.QuestionBank) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	oid, err := primitive.ObjectIDFromHex(bank.ID)
	if err != nil {
		return domain.ErrNotFound
	}
	ownerID, err := primitive.ObjectIDFromHex(bank.OwnerID)
	if err != nil {
		return domain.ErrNotFound
	}
	bank.UpdatedAt = time.Now().UTC()

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": oid, "ownerId": ownerID}, bson.M{"$set": bson.M{
		"name":        bank.Name,
		"description": bank.Description,
		"visibility":  bank.Visibility,
		"updatedAt":   bank.UpdatedAt,
	}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *QuestionBankRepository) UpdateItems(ctx context.Context, bankID string, items []domain.QuestionBankItem) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	oid, err := primitive.ObjectIDFromHex(bankID)
	if err != nil {
		return domain.ErrNotFound
	}
	itemDocs, err := toQuestionBankItemDocs(items)
	if err != nil {
		return err
	}
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": bson.M{"items": itemDocs, "updatedAt": time.Now().UTC()}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *QuestionBankRepository) UpdateShares(ctx context.Context, bankID string, shares []domain.QuestionBankShare) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	oid, err := primitive.ObjectIDFromHex(bankID)
	if err != nil {
		return domain.ErrNotFound
	}
	shareDocs, err := toQuestionBankShareDocs(shares)
	if err != nil {
		return err
	}
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": bson.M{"sharedWith": shareDocs, "updatedAt": time.Now().UTC()}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func toQuestionBankDoc(bank domain.QuestionBank) (questionBankDoc, error) {
	oid, err := primitive.ObjectIDFromHex(bank.ID)
	if err != nil {
		return questionBankDoc{}, err
	}
	ownerID, err := primitive.ObjectIDFromHex(bank.OwnerID)
	if err != nil {
		return questionBankDoc{}, err
	}
	shares, err := toQuestionBankShareDocs(bank.SharedWith)
	if err != nil {
		return questionBankDoc{}, err
	}
	items, err := toQuestionBankItemDocs(bank.Items)
	if err != nil {
		return questionBankDoc{}, err
	}
	return questionBankDoc{
		ID:          oid,
		Name:        bank.Name,
		OwnerID:     ownerID,
		Description: bank.Description,
		Visibility:  bank.Visibility,
		SharedWith:  shares,
		Items:       items,
		CreatedAt:   bank.CreatedAt,
		UpdatedAt:   bank.UpdatedAt,
	}, nil
}

func toQuestionBankShareDocs(shares []domain.QuestionBankShare) ([]questionBankShareDoc, error) {
	out := make([]questionBankShareDoc, 0, len(shares))
	for _, share := range shares {
		userID, err := primitive.ObjectIDFromHex(share.UserID)
		if err != nil {
			return nil, err
		}
		grantedBy, err := primitive.ObjectIDFromHex(share.GrantedBy)
		if err != nil {
			return nil, err
		}
		out = append(out, questionBankShareDoc{
			UserID:     userID,
			Permission: share.Permission,
			GrantedBy:  grantedBy,
			GrantedAt:  share.GrantedAt,
			ExpiresAt:  share.ExpiresAt,
		})
	}
	return out, nil
}

func toQuestionBankItemDocs(items []domain.QuestionBankItem) ([]questionBankItemDoc, error) {
	out := make([]questionBankItemDoc, 0, len(items))
	for _, item := range items {
		questionID, err := primitive.ObjectIDFromHex(item.QuestionID)
		if err != nil {
			return nil, err
		}
		addedBy, err := primitive.ObjectIDFromHex(item.AddedBy)
		if err != nil {
			return nil, err
		}
		var pinnedVersionID *primitive.ObjectID
		if item.PinnedVersionID != nil && *item.PinnedVersionID != "" {
			parsed, err := primitive.ObjectIDFromHex(*item.PinnedVersionID)
			if err != nil {
				return nil, err
			}
			pinnedVersionID = &parsed
		}
		out = append(out, questionBankItemDoc{
			QuestionID:      questionID,
			PinnedVersionID: pinnedVersionID,
			AddedBy:         addedBy,
			AddedAt:         item.AddedAt,
			Order:           item.Order,
		})
	}
	return out, nil
}

func toDomainQuestionBank(doc questionBankDoc) domain.QuestionBank {
	shares := make([]domain.QuestionBankShare, 0, len(doc.SharedWith))
	for _, share := range doc.SharedWith {
		shares = append(shares, domain.QuestionBankShare{
			UserID:     share.UserID.Hex(),
			Permission: share.Permission,
			GrantedBy:  share.GrantedBy.Hex(),
			GrantedAt:  share.GrantedAt,
			ExpiresAt:  share.ExpiresAt,
		})
	}
	items := make([]domain.QuestionBankItem, 0, len(doc.Items))
	for _, item := range doc.Items {
		var pinnedVersionID *string
		if item.PinnedVersionID != nil {
			hex := item.PinnedVersionID.Hex()
			pinnedVersionID = &hex
		}
		items = append(items, domain.QuestionBankItem{
			QuestionID:      item.QuestionID.Hex(),
			PinnedVersionID: pinnedVersionID,
			AddedBy:         item.AddedBy.Hex(),
			AddedAt:         item.AddedAt,
			Order:           item.Order,
		})
	}
	return domain.QuestionBank{
		ID:          doc.ID.Hex(),
		Name:        doc.Name,
		OwnerID:     doc.OwnerID.Hex(),
		Description: doc.Description,
		Visibility:  doc.Visibility,
		SharedWith:  shares,
		Items:       items,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}
}
