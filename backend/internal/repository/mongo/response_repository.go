package mongo

import (
	"context"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type responseDoc struct {
	ID              primitive.ObjectID        `bson:"_id,omitempty"`
	QuestionnaireID primitive.ObjectID        `bson:"questionnaireId"`
	IsAnonymous     bool                      `bson:"isAnonymous"`
	UserID          *primitive.ObjectID       `bson:"userId,omitempty"`
	Answers         []domain.Answer           `bson:"answers"`
	SubmittedAt     time.Time                 `bson:"submittedAt"`
	Statistics      domain.ResponseStatistics `bson:"statistics,omitempty"`
}

type ResponseRepository struct {
	collection *mongo.Collection
	timeout    time.Duration
}

func NewResponseRepository(db *mongo.Database, timeout time.Duration) *ResponseRepository {
	return &ResponseRepository{
		collection: db.Collection("responses"),
		timeout:    timeout,
	}
}

func (r *ResponseRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "questionnaireId", Value: 1}, {Key: "submittedAt", Value: -1}},
			Options: options.Index().SetName("idx_responses_qid_submitted"),
		},
		{
			Keys:    bson.D{{Key: "answers.questionId", Value: 1}},
			Options: options.Index().SetName("idx_responses_answer_qid"),
		},
		{
			Keys:    bson.D{{Key: "answers.questionVersionId", Value: 1}},
			Options: options.Index().SetName("idx_responses_answer_qvid"),
		},
	}
	_, err := r.collection.Indexes().CreateMany(ctx, models)
	return err
}

func (r *ResponseRepository) Create(ctx context.Context, response *domain.SurveyResponse) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	if response.ID == "" {
		response.ID = primitive.NewObjectID().Hex()
	}
	if response.SubmittedAt.IsZero() {
		response.SubmittedAt = time.Now().UTC()
	}

	respID, err := primitive.ObjectIDFromHex(response.ID)
	if err != nil {
		return err
	}
	qid, err := primitive.ObjectIDFromHex(response.QuestionnaireID)
	if err != nil {
		return err
	}

	var userObjectID *primitive.ObjectID
	if response.UserID != nil && *response.UserID != "" {
		parsed, err := primitive.ObjectIDFromHex(*response.UserID)
		if err != nil {
			return err
		}
		userObjectID = &parsed
	}

	doc := responseDoc{
		ID:              respID,
		QuestionnaireID: qid,
		IsAnonymous:     response.IsAnonymous,
		UserID:          userObjectID,
		Answers:         response.Answers,
		SubmittedAt:     response.SubmittedAt,
		Statistics:      response.Statistics,
	}

	_, err = r.collection.InsertOne(ctx, doc)
	return err
}

func (r *ResponseRepository) ListByQuestionnaire(ctx context.Context, questionnaireID string, filter domain.ResponseListFilter) ([]domain.SurveyResponse, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	qid, err := primitive.ObjectIDFromHex(questionnaireID)
	if err != nil {
		return []domain.SurveyResponse{}, 0, nil
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

	query := bson.M{"questionnaireId": qid}
	if filter.QuestionID != "" || filter.QuestionVersionID != "" {
		elemMatch := bson.M{}
		if filter.QuestionID != "" {
			elemMatch["questionId"] = filter.QuestionID
		}
		if filter.QuestionVersionID != "" {
			elemMatch["questionVersionId"] = filter.QuestionVersionID
		}
		query["answers"] = bson.M{"$elemMatch": elemMatch}
	}

	count, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSkip(int64((filter.Page - 1) * filter.Limit)).
		SetLimit(int64(filter.Limit)).
		SetSort(bson.D{{Key: "submittedAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	items := make([]domain.SurveyResponse, 0)
	for cursor.Next(ctx) {
		var doc responseDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, 0, err
		}
		items = append(items, toDomainResponse(doc))
	}
	if err := cursor.Err(); err != nil {
		return nil, 0, err
	}

	return items, count, nil
}

func (r *ResponseRepository) CountByQuestionnaire(ctx context.Context, questionnaireID string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	qid, err := primitive.ObjectIDFromHex(questionnaireID)
	if err != nil {
		return 0, nil
	}
	return r.collection.CountDocuments(ctx, bson.M{"questionnaireId": qid})
}

func toDomainResponse(doc responseDoc) domain.SurveyResponse {
	var userID *string
	if doc.UserID != nil {
		hex := doc.UserID.Hex()
		userID = &hex
	}

	return domain.SurveyResponse{
		ID:              doc.ID.Hex(),
		QuestionnaireID: doc.QuestionnaireID.Hex(),
		IsAnonymous:     doc.IsAnonymous,
		UserID:          userID,
		Answers:         doc.Answers,
		SubmittedAt:     doc.SubmittedAt,
		Statistics:      doc.Statistics,
	}
}
