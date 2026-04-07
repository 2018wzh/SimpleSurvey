package migration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type StartupValidationResult struct {
	NeedsMigration bool
	Migrated       bool
	Migration      QuestionVersionMigrationResult
}

func EnsureSchemaAtStartup(ctx context.Context, db *mongo.Database, timeout time.Duration) (StartupValidationResult, error) {
	result := StartupValidationResult{}
	if db == nil {
		return result, fmt.Errorf("database is nil")
	}

	needsMigration, err := detectV1Data(ctx, db)
	if err != nil {
		return result, err
	}
	result.NeedsMigration = needsMigration

	if needsMigration {
		migrator := NewQuestionVersionMigrator(db, timeout)
		migrationResult, err := migrator.Migrate(ctx, false)
		if err != nil {
			return result, fmt.Errorf("migrate from v1.0 failed: %w", err)
		}
		result.Migrated = true
		result.Migration = migrationResult
	}

	if err := validateStrictQuestionnaireFields(ctx, db); err != nil {
		return result, err
	}
	if err := validateStrictResponseFields(ctx, db); err != nil {
		return result, err
	}

	return result, nil
}

func detectV1Data(ctx context.Context, db *mongo.Database) (bool, error) {
	questionnaires := db.Collection("questionnaires")
	responses := db.Collection("responses")

	if has, err := collectionHasAny(ctx, questionnaires, v1QuestionnaireFilter()); err != nil {
		return false, fmt.Errorf("detect v1 questionnaires failed: %w", err)
	} else if has {
		return true, nil
	}

	if has, err := collectionHasAny(ctx, responses, v1ResponseFilter()); err != nil {
		return false, fmt.Errorf("detect v1 responses failed: %w", err)
	} else if has {
		return true, nil
	}

	return false, nil
}

func validateStrictQuestionnaireFields(ctx context.Context, db *mongo.Database) error {
	questionnaires := db.Collection("questionnaires")

	var doc bson.M
	err := questionnaires.FindOne(ctx, invalidQuestionnaireFilter()).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil
	}
	if err != nil {
		return fmt.Errorf("validate questionnaires fields failed: %w", err)
	}

	id := strings.TrimSpace(objectIDHexFromAny(doc["_id"]))
	if id == "" {
		id = "unknown"
	}
	return fmt.Errorf("questionnaire %s has invalid question refs: require questionId/questionVersionId/snapshot", id)
}

func validateStrictResponseFields(ctx context.Context, db *mongo.Database) error {
	responses := db.Collection("responses")

	var doc bson.M
	err := responses.FindOne(ctx, invalidResponseFilter()).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil
	}
	if err != nil {
		return fmt.Errorf("validate responses fields failed: %w", err)
	}

	id := strings.TrimSpace(objectIDHexFromAny(doc["_id"]))
	if id == "" {
		id = "unknown"
	}
	return fmt.Errorf("response %s has invalid answers: require questionId/questionVersionId", id)
}

func collectionHasAny(ctx context.Context, collection *mongo.Collection, filter bson.M) (bool, error) {
	if collection == nil {
		return false, fmt.Errorf("collection is nil")
	}
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func v1QuestionnaireFilter() bson.M {
	return bson.M{
		"questions": bson.M{
			"$elemMatch": bson.M{
				"$or": []bson.M{
					{"questionVersionId": bson.M{"$exists": false}},
					{"questionVersionId": ""},
					{"questionVersionId": nil},
					{"snapshot": bson.M{"$exists": false}},
					{"snapshot": nil},
				},
			},
		},
	}
}

func v1ResponseFilter() bson.M {
	return bson.M{
		"answers": bson.M{
			"$elemMatch": bson.M{
				"$or": []bson.M{
					{"questionVersionId": bson.M{"$exists": false}},
					{"questionVersionId": ""},
					{"questionVersionId": nil},
				},
			},
		},
	}
}

func invalidQuestionnaireFilter() bson.M {
	return bson.M{
		"questions": bson.M{
			"$elemMatch": bson.M{
				"$or": []bson.M{
					{"questionId": bson.M{"$exists": false}},
					{"questionId": ""},
					{"questionId": nil},
					{"questionVersionId": bson.M{"$exists": false}},
					{"questionVersionId": ""},
					{"questionVersionId": nil},
					{"snapshot": bson.M{"$exists": false}},
					{"snapshot": nil},
				},
			},
		},
	}
}

func invalidResponseFilter() bson.M {
	return bson.M{
		"answers": bson.M{
			"$elemMatch": bson.M{
				"$or": []bson.M{
					{"questionId": bson.M{"$exists": false}},
					{"questionId": ""},
					{"questionId": nil},
					{"questionVersionId": bson.M{"$exists": false}},
					{"questionVersionId": ""},
					{"questionVersionId": nil},
				},
			},
		},
	}
}
