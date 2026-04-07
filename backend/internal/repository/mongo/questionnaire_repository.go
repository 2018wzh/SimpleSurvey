package mongo

import (
	"context"
	"errors"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type questionnaireDoc struct {
	ID          primitive.ObjectID           `bson:"_id,omitempty"`
	CreatorID   primitive.ObjectID           `bson:"creatorId"`
	Title       string                       `bson:"title"`
	Description string                       `bson:"description"`
	Settings    domain.QuestionnaireSettings `bson:"settings"`
	Questions   []domain.Question            `bson:"questions"`
	LogicRules  []domain.LogicRule           `bson:"logicRules,omitempty"`
	Status      domain.QuestionnaireStatus   `bson:"status"`
	Deadline    *time.Time                   `bson:"deadline,omitempty"`
	CreatedAt   time.Time                    `bson:"createdAt"`
	UpdatedAt   time.Time                    `bson:"updatedAt"`
	IsDeleted   bool                         `bson:"isDeleted"`
}

type QuestionnaireRepository struct {
	collection *mongo.Collection
	timeout    time.Duration
}

func NewQuestionnaireRepository(db *mongo.Database, timeout time.Duration) *QuestionnaireRepository {
	return &QuestionnaireRepository{
		collection: db.Collection("questionnaires"),
		timeout:    timeout,
	}
}

func (r *QuestionnaireRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "creatorId", Value: 1}, {Key: "status", Value: 1}, {Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_questionnaires_owner_status_created"),
		},
		{
			Keys:    bson.D{{Key: "isDeleted", Value: 1}},
			Options: options.Index().SetName("idx_questionnaires_is_deleted"),
		},
		{
			Keys:    bson.D{{Key: "questions.questionVersionId", Value: 1}},
			Options: options.Index().SetName("idx_questionnaires_question_version"),
		},
	}
	_, err := r.collection.Indexes().CreateMany(ctx, models)
	return err
}

func (r *QuestionnaireRepository) Create(ctx context.Context, questionnaire *domain.Questionnaire) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	now := time.Now().UTC()
	if questionnaire.ID == "" {
		questionnaire.ID = primitive.NewObjectID().Hex()
	}
	if questionnaire.CreatedAt.IsZero() {
		questionnaire.CreatedAt = now
	}
	questionnaire.UpdatedAt = now

	qid, err := primitive.ObjectIDFromHex(questionnaire.ID)
	if err != nil {
		return err
	}
	creatorID, err := primitive.ObjectIDFromHex(questionnaire.CreatorID)
	if err != nil {
		return err
	}

	doc := questionnaireDoc{
		ID:          qid,
		CreatorID:   creatorID,
		Title:       questionnaire.Title,
		Description: questionnaire.Description,
		Settings:    questionnaire.Settings,
		Questions:   questionnaire.Questions,
		LogicRules:  questionnaire.LogicRules,
		Status:      questionnaire.Status,
		Deadline:    questionnaire.Deadline,
		CreatedAt:   questionnaire.CreatedAt,
		UpdatedAt:   questionnaire.UpdatedAt,
		IsDeleted:   questionnaire.IsDeleted,
	}

	_, err = r.collection.InsertOne(ctx, doc)
	return err
}

func (r *QuestionnaireRepository) FindByID(ctx context.Context, id string) (*domain.Questionnaire, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	qid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	var doc questionnaireDoc
	err = r.collection.FindOne(ctx, bson.M{"_id": qid, "isDeleted": false}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	questionnaire := toDomainQuestionnaire(doc)
	return &questionnaire, nil
}

func (r *QuestionnaireRepository) FindByIDAndCreator(ctx context.Context, id, creatorID string) (*domain.Questionnaire, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	qid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	ownerID, err := primitive.ObjectIDFromHex(creatorID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	var doc questionnaireDoc
	err = r.collection.FindOne(ctx, bson.M{"_id": qid, "creatorId": ownerID, "isDeleted": false}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	questionnaire := toDomainQuestionnaire(doc)
	return &questionnaire, nil
}

func (r *QuestionnaireRepository) ListByCreator(ctx context.Context, creatorID string, filter domain.QuestionnaireListFilter) ([]domain.Questionnaire, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	ownerID, err := primitive.ObjectIDFromHex(creatorID)
	if err != nil {
		return []domain.Questionnaire{}, 0, nil
	}

	query := bson.M{"creatorId": ownerID, "isDeleted": false}
	if filter.Status != "" {
		query["status"] = filter.Status
	}

	return r.listByQuery(ctx, query, filter.Page, filter.Limit, filter.SortBy)
}

func (r *QuestionnaireRepository) ListAll(ctx context.Context, filter domain.QuestionnaireAdminListFilter) ([]domain.Questionnaire, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	query := bson.M{"isDeleted": false}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.CreatorID != "" {
		ownerID, err := primitive.ObjectIDFromHex(filter.CreatorID)
		if err != nil {
			return []domain.Questionnaire{}, 0, nil
		}
		query["creatorId"] = ownerID
	}

	return r.listByQuery(ctx, query, filter.Page, filter.Limit, filter.SortBy)
}

func (r *QuestionnaireRepository) listByQuery(ctx context.Context, query bson.M, page, limit int, sortBy string) ([]domain.Questionnaire, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	count, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	sortField := "createdAt"
	if sortBy == "updatedAt" {
		sortField = "updatedAt"
	}

	opts := options.Find().
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: sortField, Value: -1}})

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	items := make([]domain.Questionnaire, 0)
	for cursor.Next(ctx) {
		var doc questionnaireDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, 0, err
		}
		items = append(items, toDomainQuestionnaire(doc))
	}
	if err := cursor.Err(); err != nil {
		return nil, 0, err
	}

	return items, count, nil
}

func (r *QuestionnaireRepository) UpdateStatus(ctx context.Context, id, creatorID string, status domain.QuestionnaireStatus, deadline *time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	qid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrNotFound
	}
	ownerID, err := primitive.ObjectIDFromHex(creatorID)
	if err != nil {
		return domain.ErrNotFound
	}

	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"updatedAt": time.Now().UTC(),
		},
	}
	if deadline != nil {
		update["$set"].(bson.M)["deadline"] = deadline
	}

	result, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": qid, "creatorId": ownerID, "isDeleted": false},
		update,
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *QuestionnaireRepository) UpdateStatusByAdmin(ctx context.Context, id string, status domain.QuestionnaireStatus, deadline *time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	qid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrNotFound
	}

	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"updatedAt": time.Now().UTC(),
		},
	}
	if deadline != nil {
		update["$set"].(bson.M)["deadline"] = deadline
	}

	result, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": qid, "isDeleted": false},
		update,
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func toDomainQuestionnaire(doc questionnaireDoc) domain.Questionnaire {
	return domain.Questionnaire{
		ID:          doc.ID.Hex(),
		CreatorID:   doc.CreatorID.Hex(),
		Title:       doc.Title,
		Description: doc.Description,
		Settings:    doc.Settings,
		Questions:   doc.Questions,
		LogicRules:  doc.LogicRules,
		Status:      doc.Status,
		Deadline:    doc.Deadline,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
		IsDeleted:   doc.IsDeleted,
	}
}
