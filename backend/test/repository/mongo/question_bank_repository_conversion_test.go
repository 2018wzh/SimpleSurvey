package mongo

import (
	"testing"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	mongorepo "github.com/2018wzh/SimpleSurvey/backend/internal/repository/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestQuestionBankDocConversionsRoundTrip(t *testing.T) {
	bankID := primitive.NewObjectID().Hex()
	ownerID := primitive.NewObjectID().Hex()
	sharedUser := primitive.NewObjectID().Hex()
	grantBy := primitive.NewObjectID().Hex()
	questionID := primitive.NewObjectID().Hex()
	pinnedVersionID := primitive.NewObjectID().Hex()
	addedBy := primitive.NewObjectID().Hex()
	now := time.Now().UTC().Truncate(time.Millisecond)

	bank := domain.QuestionBank{
		ID:          bankID,
		Name:        "基础题库",
		OwnerID:     ownerID,
		Description: "跨问卷复用",
		Visibility:  domain.QuestionBankVisibilityTeam,
		SharedWith: []domain.QuestionBankShare{{
			UserID:     sharedUser,
			Permission: domain.QuestionBankPermissionManage,
			GrantedBy:  grantBy,
			GrantedAt:  now,
		}},
		Items: []domain.QuestionBankItem{{
			QuestionID:      questionID,
			PinnedVersionID: &pinnedVersionID,
			AddedBy:         addedBy,
			AddedAt:         now,
			Order:           1,
		}},
		CreatedAt: now,
		UpdatedAt: now,
	}

	name, itemsLen, sharesLen, err := mongorepo.ToQuestionBankDocMetaForTest(bank)
	if err != nil {
		t.Fatalf("ToQuestionBankDocMetaForTest failed: %v", err)
	}
	if name != bank.Name || itemsLen != 1 || sharesLen != 1 {
		t.Fatalf("unexpected bank doc meta: name=%s items=%d shares=%d", name, itemsLen, sharesLen)
	}

	back, err := mongorepo.RoundTripQuestionBankForTest(bank)
	if err != nil {
		t.Fatalf("RoundTripQuestionBankForTest failed: %v", err)
	}
	if back.ID != bank.ID || back.OwnerID != bank.OwnerID || back.Visibility != bank.Visibility {
		t.Fatalf("unexpected round-trip bank: %+v", back)
	}
	if len(back.Items) != 1 || back.Items[0].QuestionID != questionID {
		t.Fatalf("unexpected round-trip items: %+v", back.Items)
	}
	if back.Items[0].PinnedVersionID == nil || *back.Items[0].PinnedVersionID != pinnedVersionID {
		t.Fatalf("unexpected round-trip pinnedVersionId: %+v", back.Items[0].PinnedVersionID)
	}
	if len(back.SharedWith) != 1 || back.SharedWith[0].UserID != sharedUser {
		t.Fatalf("unexpected round-trip shares: %+v", back.SharedWith)
	}
}

func TestQuestionBankConversionRejectsInvalidIDs(t *testing.T) {
	err := mongorepo.ValidateQuestionBankShareDocsForTest([]domain.QuestionBankShare{{
		UserID:     "bad-user-id",
		Permission: domain.QuestionBankPermissionUse,
		GrantedBy:  primitive.NewObjectID().Hex(),
		GrantedAt:  time.Now().UTC(),
	}})
	if err == nil {
		t.Fatal("expected error for invalid shared user id")
	}

	err = mongorepo.ValidateQuestionBankItemDocsForTest([]domain.QuestionBankItem{{
		QuestionID: "bad-question-id",
		AddedBy:    primitive.NewObjectID().Hex(),
		AddedAt:    time.Now().UTC(),
		Order:      1,
	}})
	if err == nil {
		t.Fatal("expected error for invalid question id")
	}
}
