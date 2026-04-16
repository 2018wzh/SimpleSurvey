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

type questionDoc struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	QuestionKey      string             `bson:"questionKey"`
	OwnerID          primitive.ObjectID `bson:"ownerId"`
	CurrentVersion   int                `bson:"currentVersion"`
	CurrentVersionID primitive.ObjectID `bson:"currentVersionId"`
	Tags             []string           `bson:"tags,omitempty"`
	CreatedAt        time.Time          `bson:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt"`
	IsArchived       bool               `bson:"isArchived"`
}

type questionVersionDoc struct {
	ID              primitive.ObjectID               `bson:"_id,omitempty"`
	QuestionID      primitive.ObjectID               `bson:"questionId"`
	Version         int                              `bson:"version"`
	ParentVersionID *primitive.ObjectID              `bson:"parentVersionId,omitempty"`
	ChangeType      domain.QuestionVersionChangeType `bson:"changeType"`
	Schema          domain.QuestionSchema            `bson:"schema"`
	CreatedBy       primitive.ObjectID               `bson:"createdBy"`
	CreatedAt       time.Time                        `bson:"createdAt"`
	Note            string                           `bson:"note,omitempty"`
}

type QuestionRepository struct {
	questions *mongo.Collection
	versions  *mongo.Collection
	timeout   time.Duration
}

func NewQuestionRepository(db *mongo.Database, timeout time.Duration) *QuestionRepository {
	return &QuestionRepository{
		questions: db.Collection("questions"),
		versions:  db.Collection("question_versions"),
		timeout:   timeout,
	}
}

func (r *QuestionRepository) EnsureIndexes(ctx context.Context) error {
	questionIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "ownerId", Value: 1}, {Key: "updatedAt", Value: -1}},
			Options: options.Index().SetName("idx_questions_owner_updated"),
		},
		{
			Keys:    bson.D{{Key: "questionKey", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uq_questions_question_key"),
		},
	}
	if _, err := r.questions.Indexes().CreateMany(ctx, questionIndexes); err != nil {
		return err
	}

	versionIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "questionId", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uq_question_versions_qid_version"),
		},
		{
			Keys:    bson.D{{Key: "parentVersionId", Value: 1}},
			Options: options.Index().SetName("idx_question_versions_parent"),
		},
	}
	_, err := r.versions.Indexes().CreateMany(ctx, versionIndexes)
	return err
}

func (r *QuestionRepository) Create(ctx context.Context, question *domain.QuestionEntity, version *domain.QuestionVersion) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	now := time.Now().UTC()
	if question.ID == "" {
		question.ID = primitive.NewObjectID().Hex()
	}
	if question.CreatedAt.IsZero() {
		question.CreatedAt = now
	}
	question.UpdatedAt = now
	if question.CurrentVersion == 0 {
		question.CurrentVersion = 1
	}
	if version.ID == "" {
		version.ID = primitive.NewObjectID().Hex()
	}
	if version.Version == 0 {
		version.Version = 1
	}
	if version.CreatedAt.IsZero() {
		version.CreatedAt = now
	}
	question.CurrentVersionID = version.ID
	version.QuestionID = question.ID

	qDoc, err := toQuestionDoc(*question)
	if err != nil {
		return err
	}
	vDoc, err := toQuestionVersionDoc(*version)
	if err != nil {
		return err
	}

	if _, err := r.questions.InsertOne(ctx, qDoc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrDuplicate
		}
		return err
	}
	if _, err := r.versions.InsertOne(ctx, vDoc); err != nil {
		return err
	}
	return nil
}

func (r *QuestionRepository) FindByID(ctx context.Context, id string) (*domain.QuestionEntity, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	oqid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	var doc questionDoc
	if err := r.questions.FindOne(ctx, bson.M{"_id": oqid}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	entity := toDomainQuestion(doc)
	return &entity, nil
}

func (r *QuestionRepository) FindByIDAndOwner(ctx context.Context, id, ownerID string) (*domain.QuestionEntity, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	oqid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	ooid, err := primitive.ObjectIDFromHex(ownerID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	var doc questionDoc
	if err := r.questions.FindOne(ctx, bson.M{"_id": oqid, "ownerId": ooid}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	entity := toDomainQuestion(doc)
	return &entity, nil
}

func (r *QuestionRepository) FindByQuestionKey(ctx context.Context, questionKey string) (*domain.QuestionEntity, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	var doc questionDoc
	if err := r.questions.FindOne(ctx, bson.M{"questionKey": questionKey}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	entity := toDomainQuestion(doc)
	return &entity, nil
}

func (r *QuestionRepository) FindVersionByID(ctx context.Context, versionID string) (*domain.QuestionVersion, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	ovid, err := primitive.ObjectIDFromHex(versionID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	var doc questionVersionDoc
	if err := r.versions.FindOne(ctx, bson.M{"_id": ovid}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	version := toDomainQuestionVersion(doc)
	return &version, nil
}

func (r *QuestionRepository) ListVersions(ctx context.Context, questionID string) ([]domain.QuestionVersion, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	oqid, err := primitive.ObjectIDFromHex(questionID)
	if err != nil {
		return []domain.QuestionVersion{}, nil
	}

	cursor, err := r.versions.Find(ctx, bson.M{"questionId": oqid}, options.Find().SetSort(bson.D{{Key: "version", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	items := make([]domain.QuestionVersion, 0)
	for cursor.Next(ctx) {
		var doc questionVersionDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		items = append(items, toDomainQuestionVersion(doc))
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *QuestionRepository) ListByOwner(ctx context.Context, ownerID string, filter domain.QuestionListFilter) ([]domain.QuestionEntity, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	ooid, err := primitive.ObjectIDFromHex(ownerID)
	if err != nil {
		return []domain.QuestionEntity{}, 0, domain.ErrNotFound
	}

	query := bson.M{"ownerId": ooid}
	if filter.Keyword != "" {
		query["questionKey"] = bson.M{"$regex": filter.Keyword, "$options": "i"}
	}

	total, err := r.questions.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "updatedAt", Value: -1}}).
		SetSkip(int64((filter.Page - 1) * filter.Limit)).
		SetLimit(int64(filter.Limit))

	cursor, err := r.questions.Find(ctx, query, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	items := make([]domain.QuestionEntity, 0)
	for cursor.Next(ctx) {
		var doc questionDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, 0, err
		}
		items = append(items, toDomainQuestion(doc))
	}
	if err := cursor.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *QuestionRepository) CreateVersion(ctx context.Context, question *domain.QuestionEntity, version *domain.QuestionVersion) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	if version.ID == "" {
		version.ID = primitive.NewObjectID().Hex()
	}
	if version.CreatedAt.IsZero() {
		version.CreatedAt = time.Now().UTC()
	}
	if question.UpdatedAt.IsZero() {
		question.UpdatedAt = time.Now().UTC()
	}
	question.CurrentVersion = version.Version
	question.CurrentVersionID = version.ID

	vDoc, err := toQuestionVersionDoc(*version)
	if err != nil {
		return err
	}
	if _, err := r.versions.InsertOne(ctx, vDoc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrDuplicate
		}
		return err
	}

	oqid, err := primitive.ObjectIDFromHex(question.ID)
	if err != nil {
		return err
	}
	oCurrentVersionID, err := primitive.ObjectIDFromHex(question.CurrentVersionID)
	if err != nil {
		return err
	}
	_, err = r.questions.UpdateOne(ctx, bson.M{"_id": oqid}, bson.M{"$set": bson.M{
		"currentVersion":   question.CurrentVersion,
		"currentVersionId": oCurrentVersionID,
		"updatedAt":        question.UpdatedAt,
	}})
	return err
}

func toQuestionDoc(entity domain.QuestionEntity) (questionDoc, error) {
	id, err := primitive.ObjectIDFromHex(entity.ID)
	if err != nil {
		return questionDoc{}, err
	}
	ownerID, err := primitive.ObjectIDFromHex(entity.OwnerID)
	if err != nil {
		return questionDoc{}, err
	}
	currentVersionID, err := primitive.ObjectIDFromHex(entity.CurrentVersionID)
	if err != nil {
		return questionDoc{}, err
	}
	return questionDoc{
		ID:               id,
		QuestionKey:      entity.QuestionKey,
		OwnerID:          ownerID,
		CurrentVersion:   entity.CurrentVersion,
		CurrentVersionID: currentVersionID,
		Tags:             entity.Tags,
		CreatedAt:        entity.CreatedAt,
		UpdatedAt:        entity.UpdatedAt,
		IsArchived:       entity.IsArchived,
	}, nil
}

func toQuestionVersionDoc(version domain.QuestionVersion) (questionVersionDoc, error) {
	id, err := primitive.ObjectIDFromHex(version.ID)
	if err != nil {
		return questionVersionDoc{}, err
	}
	questionID, err := primitive.ObjectIDFromHex(version.QuestionID)
	if err != nil {
		return questionVersionDoc{}, err
	}
	createdBy, err := primitive.ObjectIDFromHex(version.CreatedBy)
	if err != nil {
		return questionVersionDoc{}, err
	}
	var parentVersionID *primitive.ObjectID
	if version.ParentVersionID != nil && *version.ParentVersionID != "" {
		parsed, err := primitive.ObjectIDFromHex(*version.ParentVersionID)
		if err != nil {
			return questionVersionDoc{}, err
		}
		parentVersionID = &parsed
	}
	return questionVersionDoc{
		ID:              id,
		QuestionID:      questionID,
		Version:         version.Version,
		ParentVersionID: parentVersionID,
		ChangeType:      version.ChangeType,
		Schema:          version.Schema,
		CreatedBy:       createdBy,
		CreatedAt:       version.CreatedAt,
		Note:            version.Note,
	}, nil
}

func toDomainQuestion(doc questionDoc) domain.QuestionEntity {
	return domain.QuestionEntity{
		ID:               doc.ID.Hex(),
		QuestionKey:      doc.QuestionKey,
		OwnerID:          doc.OwnerID.Hex(),
		CurrentVersion:   doc.CurrentVersion,
		CurrentVersionID: doc.CurrentVersionID.Hex(),
		Tags:             doc.Tags,
		CreatedAt:        doc.CreatedAt,
		UpdatedAt:        doc.UpdatedAt,
		IsArchived:       doc.IsArchived,
	}
}

func toDomainQuestionVersion(doc questionVersionDoc) domain.QuestionVersion {
	var parentVersionID *string
	if doc.ParentVersionID != nil {
		hex := doc.ParentVersionID.Hex()
		parentVersionID = &hex
	}
	return domain.QuestionVersion{
		ID:              doc.ID.Hex(),
		QuestionID:      doc.QuestionID.Hex(),
		Version:         doc.Version,
		ParentVersionID: parentVersionID,
		ChangeType:      doc.ChangeType,
		Schema:          doc.Schema,
		CreatedBy:       doc.CreatedBy.Hex(),
		CreatedAt:       doc.CreatedAt,
		Note:            doc.Note,
	}
}
