package migration

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

type QuestionRefPairForTest struct {
	QuestionID string
	VersionID  string
}

func (m *QuestionVersionMigrator) PatchQuestionRefsForTest(ctx context.Context, questionnaireID, creatorID string, questions []interface{}, dryRun bool) ([]interface{}, map[string]QuestionRefPairForTest, int, int, int, error) {
	patched, mapping, patchedCount, generatedQuestions, generatedVersions, err := m.patchQuestionRefs(ctx, questionnaireID, creatorID, questions, dryRun)
	if err != nil {
		return nil, nil, 0, 0, 0, err
	}
	out := make(map[string]QuestionRefPairForTest, len(mapping))
	for k, v := range mapping {
		out[k] = QuestionRefPairForTest{QuestionID: v.QuestionID, VersionID: v.VersionID}
	}
	return patched, out, patchedCount, generatedQuestions, generatedVersions, nil
}

func PatchAnswerRefsForTest(questionnaireID string, answers []interface{}, mapping map[string]QuestionRefPairForTest) ([]interface{}, int) {
	inner := make(map[string]questionRefPair, len(mapping))
	for k, v := range mapping {
		inner[k] = questionRefPair{QuestionID: v.QuestionID, VersionID: v.VersionID}
	}
	return patchAnswerRefs(questionnaireID, answers, inner)
}

func BuildV1QuestionKeyForTest(questionnaireID, oldQuestionID string) string {
	return buildV1QuestionKey(questionnaireID, oldQuestionID)
}

func V1QuestionnaireFilterForTest() bson.M {
	return v1QuestionnaireFilter()
}

func InvalidResponseFilterForTest() bson.M {
	return invalidResponseFilter()
}
