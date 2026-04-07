package mongo

import (
	"testing"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	mongorepo "github.com/2018wzh/SimpleSurvey/backend/internal/repository/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestQuestionDocConversionsRoundTrip(t *testing.T) {
	qid := primitive.NewObjectID().Hex()
	ownerID := primitive.NewObjectID().Hex()
	versionID := primitive.NewObjectID().Hex()
	now := time.Now().UTC().Truncate(time.Millisecond)

	entity := domain.QuestionEntity{
		ID:               qid,
		QuestionKey:      "550e8400-e29b-41d4-a716-446655440000",
		OwnerID:          ownerID,
		CurrentVersion:   2,
		CurrentVersionID: versionID,
		Tags:             []string{"基础题", "人口统计"},
		CreatedAt:        now,
		UpdatedAt:        now,
		IsArchived:       false,
	}

	questionKey, currentVersion, err := mongorepo.ToQuestionDocMetaForTest(entity)
	if err != nil {
		t.Fatalf("ToQuestionDocMetaForTest failed: %v", err)
	}
	if questionKey != entity.QuestionKey || currentVersion != 2 {
		t.Fatalf("unexpected mapped question doc meta: key=%s version=%d", questionKey, currentVersion)
	}

	back, err := mongorepo.RoundTripQuestionForTest(entity)
	if err != nil {
		t.Fatalf("RoundTripQuestionForTest failed: %v", err)
	}
	if back.ID != entity.ID || back.OwnerID != entity.OwnerID || back.CurrentVersionID != entity.CurrentVersionID {
		t.Fatalf("unexpected round-trip entity: %+v", back)
	}
}

func TestQuestionVersionDocConversionsWithParent(t *testing.T) {
	vid := primitive.NewObjectID().Hex()
	qid := primitive.NewObjectID().Hex()
	creatorID := primitive.NewObjectID().Hex()
	parent := primitive.NewObjectID().Hex()
	now := time.Now().UTC().Truncate(time.Millisecond)

	version := domain.QuestionVersion{
		ID:              vid,
		QuestionID:      qid,
		Version:         3,
		ParentVersionID: &parent,
		ChangeType:      domain.QuestionVersionChangeTypeRestore,
		Schema:          domain.QuestionSchema{Type: domain.QuestionTypeText, Title: "建议"},
		CreatedBy:       creatorID,
		CreatedAt:       now,
		Note:            "恢复历史版本",
	}

	back, err := mongorepo.RoundTripQuestionVersionForTest(version)
	if err != nil {
		t.Fatalf("RoundTripQuestionVersionForTest failed: %v", err)
	}

	if back.ID != version.ID || back.QuestionID != version.QuestionID || back.ChangeType != version.ChangeType {
		t.Fatalf("unexpected round-trip version: %+v", back)
	}
	if back.ParentVersionID == nil || *back.ParentVersionID != parent {
		t.Fatalf("unexpected round-trip parent: %+v", back.ParentVersionID)
	}
}

func TestQuestionConversionsRejectInvalidIDs(t *testing.T) {
	err := mongorepo.ValidateToQuestionDocForTest(domain.QuestionEntity{ID: "bad", OwnerID: primitive.NewObjectID().Hex(), CurrentVersionID: primitive.NewObjectID().Hex()})
	if err == nil {
		t.Fatal("expected error for invalid question id")
	}

	err = mongorepo.ValidateToQuestionVersionDocForTest(domain.QuestionVersion{ID: primitive.NewObjectID().Hex(), QuestionID: primitive.NewObjectID().Hex(), CreatedBy: "bad"})
	if err == nil {
		t.Fatal("expected error for invalid createdBy")
	}
}
