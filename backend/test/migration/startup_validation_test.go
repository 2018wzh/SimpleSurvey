package migration

import (
	"testing"

	migrationpkg "github.com/2018wzh/SimpleSurvey/backend/internal/migration"
	"go.mongodb.org/mongo-driver/bson"
)

func TestEnsureSchemaAtStartupNilDB(t *testing.T) {
	_, err := migrationpkg.EnsureSchemaAtStartup(t.Context(), nil, 0)
	if err == nil {
		t.Fatal("expected error when db is nil")
	}
}

func TestV1QuestionnaireFilterContainsSnapshotAndVersionChecks(t *testing.T) {
	filter := migrationpkg.V1QuestionnaireFilterForTest()
	questions, ok := filter["questions"].(bson.M)
	if !ok {
		t.Fatalf("unexpected questions filter type: %T", filter["questions"])
	}
	elem, ok := questions["$elemMatch"].(bson.M)
	if !ok {
		t.Fatalf("unexpected elemMatch type: %T", questions["$elemMatch"])
	}
	ors, ok := elem["$or"].([]bson.M)
	if !ok || len(ors) == 0 {
		t.Fatalf("expected non-empty $or predicates, got: %#v", elem["$or"])
	}
}

func TestInvalidResponseFilterContainsAnswerChecks(t *testing.T) {
	filter := migrationpkg.InvalidResponseFilterForTest()
	answers, ok := filter["answers"].(bson.M)
	if !ok {
		t.Fatalf("unexpected answers filter type: %T", filter["answers"])
	}
	elem, ok := answers["$elemMatch"].(bson.M)
	if !ok {
		t.Fatalf("unexpected elemMatch type: %T", answers["$elemMatch"])
	}
	ors, ok := elem["$or"].([]bson.M)
	if !ok || len(ors) < 4 {
		t.Fatalf("expected answer validation predicates, got: %#v", elem["$or"])
	}
}
