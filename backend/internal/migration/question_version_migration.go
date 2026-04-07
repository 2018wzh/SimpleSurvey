package migration

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type QuestionVersionMigrationResult struct {
	QuestionnairesScanned int
	QuestionnairesUpdated int
	QuestionsPatched      int
	ResponsesScanned      int
	ResponsesUpdated      int
	AnswersPatched        int
	QuestionsGenerated    int
	VersionsGenerated     int
}

type questionRefPair struct {
	QuestionID string
	VersionID  string
}

type QuestionVersionMigrator struct {
	questionnaires *mongo.Collection
	responses      *mongo.Collection
	questions      *mongo.Collection
	versions       *mongo.Collection
	timeout        time.Duration
}

func NewQuestionVersionMigrator(db *mongo.Database, timeout time.Duration) *QuestionVersionMigrator {
	return &QuestionVersionMigrator{
		questionnaires: db.Collection("questionnaires"),
		responses:      db.Collection("responses"),
		questions:      db.Collection("questions"),
		versions:       db.Collection("question_versions"),
		timeout:        timeout,
	}
}

func (m *QuestionVersionMigrator) Migrate(ctx context.Context, dryRun bool) (QuestionVersionMigrationResult, error) {
	result := QuestionVersionMigrationResult{}
	questionMappings := map[string]map[string]questionRefPair{}

	questionnaireCursor, err := m.questionnaires.Find(ctx, bson.M{})
	if err != nil {
		return result, err
	}
	defer questionnaireCursor.Close(ctx)

	for questionnaireCursor.Next(ctx) {
		var doc bson.M
		if err := questionnaireCursor.Decode(&doc); err != nil {
			return result, err
		}

		result.QuestionnairesScanned++

		questionnaireID := objectIDHexFromAny(doc["_id"])
		creatorID := objectIDHexFromAny(doc["creatorId"])
		questions := toInterfaceSlice(doc["questions"])

		patchedQuestions, mapping, patchCount, generatedQuestions, generatedVersions, err := m.patchQuestionRefs(ctx, questionnaireID, creatorID, questions, dryRun)
		if err != nil {
			return result, err
		}
		questionMappings[questionnaireID] = mapping
		result.QuestionsPatched += patchCount
		result.QuestionsGenerated += generatedQuestions
		result.VersionsGenerated += generatedVersions

		if patchCount == 0 || dryRun {
			continue
		}
		opCtx, cancel := context.WithTimeout(ctx, m.timeout)
		_, err = m.questionnaires.UpdateByID(opCtx, doc["_id"], bson.M{"$set": bson.M{"questions": patchedQuestions}})
		cancel()
		if err != nil {
			return result, err
		}
		result.QuestionnairesUpdated++
	}
	if err := questionnaireCursor.Err(); err != nil {
		return result, err
	}

	responseCursor, err := m.responses.Find(ctx, bson.M{})
	if err != nil {
		return result, err
	}
	defer responseCursor.Close(ctx)

	for responseCursor.Next(ctx) {
		var doc bson.M
		if err := responseCursor.Decode(&doc); err != nil {
			return result, err
		}

		result.ResponsesScanned++

		questionnaireID := objectIDHexFromAny(doc["questionnaireId"])
		answers := toInterfaceSlice(doc["answers"])
		mapping := questionMappings[questionnaireID]

		patchedAnswers, patchedCount := patchAnswerRefs(questionnaireID, answers, mapping)
		result.AnswersPatched += patchedCount

		if patchedCount == 0 || dryRun {
			continue
		}
		opCtx, cancel := context.WithTimeout(ctx, m.timeout)
		_, err := m.responses.UpdateByID(opCtx, doc["_id"], bson.M{"$set": bson.M{"answers": patchedAnswers}})
		cancel()
		if err != nil {
			return result, err
		}
		result.ResponsesUpdated++
	}
	if err := responseCursor.Err(); err != nil {
		return result, err
	}

	return result, nil
}

func (m *QuestionVersionMigrator) patchQuestionRefs(ctx context.Context, questionnaireID, creatorID string, questions []interface{}, dryRun bool) ([]interface{}, map[string]questionRefPair, int, int, int, error) {
	mapping := map[string]questionRefPair{}
	patched := make([]interface{}, 0, len(questions))
	patchCount := 0
	generatedQuestions := 0
	generatedVersions := 0

	for _, raw := range questions {
		q := toBsonMap(raw)
		rawQuestionID := strings.TrimSpace(asString(q["questionId"]))
		if rawQuestionID == "" {
			patched = append(patched, q)
			continue
		}

		currentQuestionVersionID := strings.TrimSpace(asString(q["questionVersionId"]))
		if isObjectIDHex(rawQuestionID) && isObjectIDHex(currentQuestionVersionID) {
			if _, hasSnapshot := q["snapshot"]; !hasSnapshot {
				q["snapshot"] = buildLegacySchema(q)
				patchCount++
			}
			mapping[rawQuestionID] = questionRefPair{QuestionID: rawQuestionID, VersionID: currentQuestionVersionID}
			patched = append(patched, q)
			continue
		}

		refPair, createdQuestion, createdVersion, err := m.ensureQuestionGenerated(ctx, creatorID, questionnaireID, rawQuestionID, q, dryRun)
		if err != nil {
			return nil, nil, 0, 0, 0, err
		}
		if createdQuestion {
			generatedQuestions++
		}
		if createdVersion {
			generatedVersions++
		}
		mapping[rawQuestionID] = refPair
		mapping[refPair.QuestionID] = refPair

		if asString(q["questionId"]) != refPair.QuestionID {
			q["questionId"] = refPair.QuestionID
			patchCount++
		}
		if strings.TrimSpace(asString(q["questionVersionId"])) != refPair.VersionID {
			q["questionVersionId"] = refPair.VersionID
			patchCount++
		}
		if _, hasSnapshot := q["snapshot"]; !hasSnapshot {
			q["snapshot"] = buildLegacySchema(q)
			patchCount++
		}
		patched = append(patched, q)
	}

	return patched, mapping, patchCount, generatedQuestions, generatedVersions, nil
}

func (m *QuestionVersionMigrator) ensureQuestionGenerated(ctx context.Context, creatorID, questionnaireID, oldQuestionID string, legacyQuestion bson.M, dryRun bool) (questionRefPair, bool, bool, error) {
	questionKey := buildV1QuestionKey(questionnaireID, oldQuestionID)

	if m.questions == nil || m.versions == nil {
		qid := primitive.NewObjectID().Hex()
		vid := primitive.NewObjectID().Hex()
		return questionRefPair{QuestionID: qid, VersionID: vid}, true, true, nil
	}

	opCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	var existingQuestion bson.M
	err := m.questions.FindOne(opCtx, bson.M{"questionKey": questionKey}).Decode(&existingQuestion)
	if err == nil {
		qid := objectIDHexFromAny(existingQuestion["_id"])
		qvid := objectIDHexFromAny(existingQuestion["currentVersionId"])
		if qid != "" && qvid != "" {
			return questionRefPair{QuestionID: qid, VersionID: qvid}, false, false, nil
		}
	}
	if err != nil && err != mongo.ErrNoDocuments {
		return questionRefPair{}, false, false, err
	}

	questionObjectID := primitive.NewObjectID()
	versionObjectID := primitive.NewObjectID()
	now := time.Now().UTC()
	ownerObjectID, ownerOK := parseObjectID(creatorID)
	if !ownerOK {
		ownerObjectID = primitive.NilObjectID
	}
	schema := buildLegacySchema(legacyQuestion)

	if !dryRun {
		questionDoc := bson.M{
			"_id":              questionObjectID,
			"questionKey":      questionKey,
			"ownerId":          ownerObjectID,
			"currentVersion":   1,
			"currentVersionId": versionObjectID,
			"tags":             primitive.A{},
			"createdAt":        now,
			"updatedAt":        now,
			"isArchived":       false,
		}
		if _, err := m.questions.InsertOne(opCtx, questionDoc); err != nil {
			if !mongo.IsDuplicateKeyError(err) {
				return questionRefPair{}, false, false, err
			}
			// another worker might have inserted; re-read and use existing
			var conflictQuestion bson.M
			if err := m.questions.FindOne(opCtx, bson.M{"questionKey": questionKey}).Decode(&conflictQuestion); err != nil {
				return questionRefPair{}, false, false, err
			}
			qid := objectIDHexFromAny(conflictQuestion["_id"])
			qvid := objectIDHexFromAny(conflictQuestion["currentVersionId"])
			if qid != "" && qvid != "" {
				return questionRefPair{QuestionID: qid, VersionID: qvid}, false, false, nil
			}
		}

		versionDoc := bson.M{
			"_id":        versionObjectID,
			"questionId": questionObjectID,
			"version":    1,
			"changeType": "create",
			"schema":     schema,
			"createdBy":  ownerObjectID,
			"createdAt":  now,
			"note":       "migrated from v1.0",
		}
		if _, err := m.versions.InsertOne(opCtx, versionDoc); err != nil {
			if !mongo.IsDuplicateKeyError(err) {
				return questionRefPair{}, true, false, err
			}
		}
	}

	return questionRefPair{QuestionID: questionObjectID.Hex(), VersionID: versionObjectID.Hex()}, true, true, nil
}

func patchAnswerRefs(questionnaireID string, answers []interface{}, questionMappings map[string]questionRefPair) ([]interface{}, int) {
	patched := make([]interface{}, 0, len(answers))
	patchedCount := 0

	for _, raw := range answers {
		ans := toBsonMap(raw)
		rawQuestionID := strings.TrimSpace(asString(ans["questionId"]))
		if rawQuestionID == "" {
			patched = append(patched, ans)
			continue
		}
		if ref, ok := questionMappings[rawQuestionID]; ok {
			if asString(ans["questionId"]) != ref.QuestionID {
				ans["questionId"] = ref.QuestionID
				patchedCount++
			}
			if strings.TrimSpace(asString(ans["questionVersionId"])) != ref.VersionID {
				ans["questionVersionId"] = ref.VersionID
				patchedCount++
			}
			patched = append(patched, ans)
			continue
		}

		questionVersionID := strings.TrimSpace(asString(ans["questionVersionId"]))
		_ = questionVersionID
		patched = append(patched, ans)
	}

	return patched, patchedCount
}

func buildV1QuestionKey(questionnaireID, oldQuestionID string) string {
	base := strings.TrimSpace(questionnaireID) + "::" + strings.TrimSpace(oldQuestionID)
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(base)).String()
}

func buildLegacySchema(question bson.M) bson.M {
	if snapshot := toBsonMap(question["snapshot"]); len(snapshot) > 0 {
		schema := bson.M{
			"type":       asString(snapshot["type"]),
			"title":      asString(snapshot["title"]),
			"isRequired": asBool(snapshot["isRequired"]),
		}
		if meta := toBsonMap(snapshot["meta"]); len(meta) > 0 {
			schema["meta"] = meta
		}
		if options := toInterfaceSlice(snapshot["options"]); len(options) > 0 {
			schema["options"] = options
		}
		if validation := toBsonMap(snapshot["validation"]); len(validation) > 0 {
			schema["validation"] = validation
		}
		return schema
	}

	schema := bson.M{
		"type":       asString(question["type"]),
		"title":      asString(question["title"]),
		"isRequired": asBool(question["isRequired"]),
	}
	if meta := toBsonMap(question["meta"]); len(meta) > 0 {
		schema["meta"] = meta
	}
	if options := toInterfaceSlice(question["options"]); len(options) > 0 {
		schema["options"] = options
	}
	if validation := toBsonMap(question["validation"]); len(validation) > 0 {
		schema["validation"] = validation
	}
	return schema
}

func toInterfaceSlice(value interface{}) []interface{} {
	switch v := value.(type) {
	case []interface{}:
		return v
	case primitive.A:
		return []interface{}(v)
	default:
		return []interface{}{}
	}
}

func toBsonMap(value interface{}) bson.M {
	switch v := value.(type) {
	case bson.M:
		return v
	case map[string]interface{}:
		return bson.M(v)
	default:
		return bson.M{}
	}
}

func asString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func asBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	default:
		return false
	}
}

func objectIDHexFromAny(value interface{}) string {
	switch v := value.(type) {
	case primitive.ObjectID:
		return v.Hex()
	case string:
		if oid, err := primitive.ObjectIDFromHex(v); err == nil {
			return oid.Hex()
		}
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func parseObjectID(value string) (primitive.ObjectID, bool) {
	oid, err := primitive.ObjectIDFromHex(strings.TrimSpace(value))
	if err != nil {
		return primitive.NilObjectID, false
	}
	return oid, true
}

func isObjectIDHex(value string) bool {
	_, err := primitive.ObjectIDFromHex(strings.TrimSpace(value))
	return err == nil
}
