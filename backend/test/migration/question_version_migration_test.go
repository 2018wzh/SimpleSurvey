package migration

import (
	"testing"

	migrationpkg "github.com/2018wzh/SimpleSurvey/backend/internal/migration"
	"go.mongodb.org/mongo-driver/bson"
)

func TestPatchQuestionVersionIDs(t *testing.T) {
	questionnaireID := "qnr-1"
	questions := []interface{}{
		bson.M{"questionId": "q1", "title": "Q1"},
		bson.M{"questionId": "q2", "questionVersionId": "ver-q2", "title": "Q2"},
	}

	m := &migrationpkg.QuestionVersionMigrator{}
	patched, mapping, count, _, _, err := m.PatchQuestionRefsForTest(t.Context(), questionnaireID, "507f1f77bcf86cd799439011", questions, true)
	if err != nil {
		t.Fatalf("unexpected patchQuestionRefs error: %v", err)
	}
	if count != 6 {
		t.Fatalf("expected patched count 6, got %d", count)
	}
	if mapping["q1"].VersionID == "" || mapping["q2"].VersionID == "" {
		t.Fatalf("unexpected mapping: %+v", mapping)
	}

	first := patched[0].(bson.M)
	if first["questionVersionId"] == "" {
		t.Fatalf("unexpected generated questionVersionId: %v", first["questionVersionId"])
	}
	if first["questionId"] == "q1" {
		t.Fatalf("expected questionId remapped from legacy id, got %v", first["questionId"])
	}
	if _, ok := first["snapshot"]; !ok {
		t.Fatalf("expected snapshot patched for migrated question")
	}
}

func TestPatchAnswerRefs(t *testing.T) {
	questionnaireID := "qnr-9"
	mapping := map[string]migrationpkg.QuestionRefPairForTest{
		"q1": {
			QuestionID: "67f3e5f244f95a7d05b5a111",
			VersionID:  "67f3e5f244f95a7d05b5a211",
		},
	}
	answers := []interface{}{
		bson.M{"questionId": "q1", "value": "A"},
		bson.M{"questionId": "q2", "value": "B"},
		bson.M{"questionId": "q3", "questionVersionId": "ver-q3", "value": "C"},
	}

	patched, count := migrationpkg.PatchAnswerRefsForTest(questionnaireID, answers, mapping)
	if count != 2 {
		t.Fatalf("expected patched count 2, got %d", count)
	}

	ans1 := patched[0].(bson.M)
	if ans1["questionVersionId"] != "67f3e5f244f95a7d05b5a211" {
		t.Fatalf("expected mapped version, got %v", ans1["questionVersionId"])
	}
	if ans1["questionId"] != "67f3e5f244f95a7d05b5a111" {
		t.Fatalf("expected mapped question id, got %v", ans1["questionId"])
	}

	ans2 := patched[1].(bson.M)
	if _, exists := ans2["questionVersionId"]; exists {
		t.Fatalf("expected no fallback version for unmapped answer, got %v", ans2["questionVersionId"])
	}
}

func TestPatchQuestionVersionIDsNoopWhenAllUpToDate(t *testing.T) {
	questionnaireID := "qnr-2"
	questions := []interface{}{
		bson.M{"questionId": "67f3e5f244f95a7d05b5a111", "questionVersionId": "67f3e5f244f95a7d05b5a211", "title": "Q1"},
		bson.M{"questionId": "67f3e5f244f95a7d05b5a112", "questionVersionId": "67f3e5f244f95a7d05b5a212", "title": "Q2"},
	}

	m := &migrationpkg.QuestionVersionMigrator{}
	patched, mapping, count, _, _, err := m.PatchQuestionRefsForTest(t.Context(), questionnaireID, "507f1f77bcf86cd799439011", questions, true)
	if err != nil {
		t.Fatalf("unexpected patchQuestionRefs error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected patched count 2 for snapshot backfill, got %d", count)
	}
	if mapping["67f3e5f244f95a7d05b5a111"].VersionID != "67f3e5f244f95a7d05b5a211" || mapping["67f3e5f244f95a7d05b5a112"].VersionID != "67f3e5f244f95a7d05b5a212" {
		t.Fatalf("unexpected mapping: %+v", mapping)
	}
	if patched[0].(bson.M)["questionVersionId"] != "67f3e5f244f95a7d05b5a211" {
		t.Fatalf("expected keep existing version, got %v", patched[0].(bson.M)["questionVersionId"])
	}
	if _, ok := patched[0].(bson.M)["snapshot"]; !ok {
		t.Fatalf("expected snapshot patched for up-to-date refs")
	}
}

func TestBuildV1QuestionKeyStable(t *testing.T) {
	k1 := migrationpkg.BuildV1QuestionKeyForTest("qnr-1", "legacy-q1")
	k2 := migrationpkg.BuildV1QuestionKeyForTest("qnr-1", "legacy-q1")
	k3 := migrationpkg.BuildV1QuestionKeyForTest("qnr-2", "legacy-q1")
	if k1 != k2 {
		t.Fatalf("expected deterministic key, got %s and %s", k1, k2)
	}
	if k1 == k3 {
		t.Fatalf("expected different questionnaire to produce different key")
	}
}
