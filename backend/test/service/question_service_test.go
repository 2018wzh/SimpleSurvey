package service

import (
	"context"
	"testing"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	. "github.com/2018wzh/SimpleSurvey/backend/internal/service"
	"github.com/google/uuid"
)

func mustQuestionSchemaSingleChoice(title string) domain.QuestionSchema {
	return domain.QuestionSchema{
		Type:       domain.QuestionTypeSingleChoice,
		Title:      title,
		IsRequired: true,
		Options:    []domain.QuestionOption{{OptionID: "a", Text: "A"}, {OptionID: "b", Text: "B"}},
	}
}

type fakeQuestionRepo struct {
	questionsByID map[string]*domain.QuestionEntity
	questionByKey map[string]*domain.QuestionEntity
	versionsByID  map[string]*domain.QuestionVersion
	versionsByQID map[string][]domain.QuestionVersion
}

func newFakeQuestionRepo() *fakeQuestionRepo {
	return &fakeQuestionRepo{
		questionsByID: map[string]*domain.QuestionEntity{},
		questionByKey: map[string]*domain.QuestionEntity{},
		versionsByID:  map[string]*domain.QuestionVersion{},
		versionsByQID: map[string][]domain.QuestionVersion{},
	}
}

func (f *fakeQuestionRepo) Create(_ context.Context, question *domain.QuestionEntity, version *domain.QuestionVersion) error {
	if question.ID == "" {
		question.ID = "67f3e5f244f95a7d05b5a111"
	}
	if version.ID == "" {
		version.ID = "67f3e5f244f95a7d05b5a211"
	}
	version.QuestionID = question.ID
	question.CurrentVersionID = version.ID
	qCopy := *question
	vCopy := *version
	f.questionsByID[qCopy.ID] = &qCopy
	f.questionByKey[qCopy.QuestionKey] = &qCopy
	f.versionsByID[vCopy.ID] = &vCopy
	f.versionsByQID[qCopy.ID] = append(f.versionsByQID[qCopy.ID], vCopy)
	return nil
}

func (f *fakeQuestionRepo) FindByID(_ context.Context, id string) (*domain.QuestionEntity, error) {
	q, ok := f.questionsByID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := *q
	return &copy, nil
}

func (f *fakeQuestionRepo) FindByIDAndOwner(_ context.Context, id, ownerID string) (*domain.QuestionEntity, error) {
	q, ok := f.questionsByID[id]
	if !ok || q.OwnerID != ownerID {
		return nil, domain.ErrNotFound
	}
	copy := *q
	return &copy, nil
}

func (f *fakeQuestionRepo) FindByQuestionKey(_ context.Context, questionKey string) (*domain.QuestionEntity, error) {
	q, ok := f.questionByKey[questionKey]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := *q
	return &copy, nil
}

func (f *fakeQuestionRepo) FindVersionByID(_ context.Context, versionID string) (*domain.QuestionVersion, error) {
	v, ok := f.versionsByID[versionID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := *v
	return &copy, nil
}

func (f *fakeQuestionRepo) ListVersions(_ context.Context, questionID string) ([]domain.QuestionVersion, error) {
	items := f.versionsByQID[questionID]
	out := make([]domain.QuestionVersion, len(items))
	copy(out, items)
	return out, nil
}

func (f *fakeQuestionRepo) CreateVersion(_ context.Context, question *domain.QuestionEntity, version *domain.QuestionVersion) error {
	if version.ID == "" {
		version.ID = "67f3e5f244f95a7d05b5a299"
	}
	version.QuestionID = question.ID
	question.CurrentVersion = version.Version
	question.CurrentVersionID = version.ID
	qCopy := *question
	vCopy := *version
	f.questionsByID[qCopy.ID] = &qCopy
	f.questionByKey[qCopy.QuestionKey] = &qCopy
	f.versionsByID[vCopy.ID] = &vCopy
	f.versionsByQID[qCopy.ID] = append(f.versionsByQID[qCopy.ID], vCopy)
	return nil
}

func TestQuestionServiceCreateAndVersionFlow(t *testing.T) {
	qRepo := newFakeQuestionRepo()
	qnRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionService(qRepo, qnRepo, rRepo)

	key := uuid.NewString()
	createResult, appErr := svc.Create(context.Background(), "507f1f77bcf86cd799439011", CreateQuestionInput{
		QuestionKey: key,
		Schema: domain.QuestionSchema{
			Type:       domain.QuestionTypeSingleChoice,
			Title:      "是否满意",
			IsRequired: true,
			Options:    []domain.QuestionOption{{OptionID: "a", Text: "是"}, {OptionID: "b", Text: "否"}},
		},
		Tags: []string{"体验", "体验"},
	})
	if appErr != nil {
		t.Fatalf("expected create success, got appErr=%v", appErr)
	}
	if createResult.Version != 1 || createResult.ID == "" || createResult.VersionID == "" {
		t.Fatalf("unexpected create result: %+v", createResult)
	}

	newVersionResult, appErr := svc.CreateVersion(context.Background(), "507f1f77bcf86cd799439011", createResult.ID, CreateQuestionVersionInput{
		BaseVersionID: createResult.VersionID,
		ChangeType:    domain.QuestionVersionChangeTypeEdit,
		Note:          "微调文案",
		Schema: domain.QuestionSchema{
			Type:       domain.QuestionTypeSingleChoice,
			Title:      "是否非常满意",
			IsRequired: true,
			Options:    []domain.QuestionOption{{OptionID: "a", Text: "是"}, {OptionID: "b", Text: "否"}},
		},
	})
	if appErr != nil {
		t.Fatalf("expected create version success, got appErr=%v", appErr)
	}
	if newVersionResult.Version != 2 {
		t.Fatalf("expected version 2, got %d", newVersionResult.Version)
	}

	versions, appErr := svc.ListVersions(context.Background(), "507f1f77bcf86cd799439011", createResult.ID)
	if appErr != nil {
		t.Fatalf("expected list versions success, got appErr=%v", appErr)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
}

func TestQuestionServiceUsagesAndStats(t *testing.T) {
	qRepo := newFakeQuestionRepo()
	qnRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionService(qRepo, qnRepo, rRepo)

	qRepo.questionsByID["qid-1"] = &domain.QuestionEntity{ID: "qid-1", QuestionKey: uuid.NewString(), OwnerID: "507f1f77bcf86cd799439011", CurrentVersion: 1, CurrentVersionID: "qv-1"}
	qRepo.versionsByID["qv-1"] = &domain.QuestionVersion{ID: "qv-1", QuestionID: "qid-1", Version: 1, Schema: domain.QuestionSchema{Type: domain.QuestionTypeSingleChoice, Title: "满意度"}}

	qnRepo.items["qn-1"] = domain.Questionnaire{
		ID:        "qn-1",
		Title:     "问卷A",
		Status:    domain.QuestionnaireStatusPublished,
		Questions: []domain.Question{{QuestionID: "qid-1", QuestionVersionID: "qv-1", Type: domain.QuestionTypeSingleChoice}},
	}
	rRepo.items["qn-1"] = []domain.SurveyResponse{{
		QuestionnaireID: "qn-1",
		SubmittedAt:     time.Now().UTC(),
		Answers:         []domain.Answer{{QuestionID: "qid-1", QuestionVersionID: "qv-1", Value: "a"}, {QuestionID: "qid-1", QuestionVersionID: "qv-1", Value: "a"}},
	}}

	usages, appErr := svc.GetUsages(context.Background(), "qid-1", "", "")
	if appErr != nil {
		t.Fatalf("expected usages success, got appErr=%v", appErr)
	}
	if len(usages) != 1 || usages[0].QuestionnaireID != "qn-1" {
		t.Fatalf("unexpected usages: %+v", usages)
	}

	stats, appErr := svc.GetStats(context.Background(), "qid-1", QuestionStatsInput{QuestionVersionID: "qv-1"})
	if appErr != nil {
		t.Fatalf("expected stats success, got appErr=%v", appErr)
	}
	if stats.TotalAnswered != 2 || stats.OptionCounts["a"] != 2 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestQuestionServiceRestoreVersionAndCoexistence(t *testing.T) {
	qRepo := newFakeQuestionRepo()
	qnRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionService(qRepo, qnRepo, rRepo)

	ownerID := "507f1f77bcf86cd799439011"
	created, appErr := svc.Create(context.Background(), ownerID, CreateQuestionInput{
		QuestionKey: uuid.NewString(),
		Schema:      mustQuestionSchemaSingleChoice("满意度v1"),
	})
	if appErr != nil {
		t.Fatalf("expected create success, got appErr=%v", appErr)
	}

	v2, appErr := svc.CreateVersion(context.Background(), ownerID, created.ID, CreateQuestionVersionInput{
		BaseVersionID: created.VersionID,
		ChangeType:    domain.QuestionVersionChangeTypeEdit,
		Schema:        mustQuestionSchemaSingleChoice("满意度v2"),
	})
	if appErr != nil {
		t.Fatalf("expected create version success, got appErr=%v", appErr)
	}

	qnRepo.items["qn-old"] = domain.Questionnaire{
		ID:     "qn-old",
		Title:  "旧问卷",
		Status: domain.QuestionnaireStatusPublished,
		Questions: []domain.Question{{
			QuestionID:        created.ID,
			QuestionVersionID: created.VersionID,
			Type:              domain.QuestionTypeSingleChoice,
		}},
	}
	qnRepo.items["qn-new"] = domain.Questionnaire{
		ID:     "qn-new",
		Title:  "新问卷",
		Status: domain.QuestionnaireStatusDraft,
		Questions: []domain.Question{{
			QuestionID:        created.ID,
			QuestionVersionID: v2.VersionID,
			Type:              domain.QuestionTypeSingleChoice,
		}},
	}

	restored, appErr := svc.RestoreVersion(context.Background(), ownerID, created.ID, RestoreQuestionVersionInput{
		FromVersionID: created.VersionID,
		Note:          "回滚到v1",
	})
	if appErr != nil {
		t.Fatalf("expected restore success, got appErr=%v", appErr)
	}
	if restored.Version != 3 {
		t.Fatalf("expected restored version=3, got %d", restored.Version)
	}

	versions, appErr := svc.ListVersions(context.Background(), ownerID, created.ID)
	if appErr != nil {
		t.Fatalf("expected list versions success, got appErr=%v", appErr)
	}
	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}
	latest := versions[len(versions)-1]
	if latest.ChangeType != domain.QuestionVersionChangeTypeRestore {
		t.Fatalf("expected latest changeType=restore, got %s", latest.ChangeType)
	}
	if latest.ParentVersionID == nil || *latest.ParentVersionID != created.VersionID {
		t.Fatalf("expected restore parentVersionId=%s, got %+v", created.VersionID, latest.ParentVersionID)
	}

	usagesV1, appErr := svc.GetUsages(context.Background(), created.ID, created.VersionID, "")
	if appErr != nil {
		t.Fatalf("expected usages by version success, got appErr=%v", appErr)
	}
	if len(usagesV1) != 1 || usagesV1[0].QuestionnaireID != "qn-old" {
		t.Fatalf("expected qn-old only for v1 usages, got %+v", usagesV1)
	}

	publishedUsages, appErr := svc.GetUsages(context.Background(), created.ID, "", string(domain.QuestionnaireStatusPublished))
	if appErr != nil {
		t.Fatalf("expected usages by status success, got appErr=%v", appErr)
	}
	if len(publishedUsages) != 1 || publishedUsages[0].QuestionnaireID != "qn-old" {
		t.Fatalf("expected qn-old only for published usages, got %+v", publishedUsages)
	}
}

func TestQuestionServiceCreateVersionRejectForeignBaseVersion(t *testing.T) {
	qRepo := newFakeQuestionRepo()
	qnRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionService(qRepo, qnRepo, rRepo)

	ownerID := "507f1f77bcf86cd799439011"
	first, appErr := svc.Create(context.Background(), ownerID, CreateQuestionInput{
		QuestionKey: uuid.NewString(),
		Schema:      mustQuestionSchemaSingleChoice("问题一"),
	})
	if appErr != nil {
		t.Fatalf("expected create first question success, got appErr=%v", appErr)
	}

	qRepo.versionsByID["foreign-v1"] = &domain.QuestionVersion{
		ID:         "foreign-v1",
		QuestionID: "another-question-id",
		Version:    1,
		Schema:     mustQuestionSchemaSingleChoice("外部问题版本"),
	}

	_, appErr = svc.CreateVersion(context.Background(), ownerID, first.ID, CreateQuestionVersionInput{
		BaseVersionID: "foreign-v1",
		ChangeType:    domain.QuestionVersionChangeTypeEdit,
		Schema:        mustQuestionSchemaSingleChoice("非法基线版本"),
	})
	if appErr == nil || appErr.Code != 400 {
		t.Fatalf("expected bad request for foreign baseVersionId, got %+v", appErr)
	}
}

func TestQuestionServiceCreateRejectsNonUUIDQuestionKey(t *testing.T) {
	qRepo := newFakeQuestionRepo()
	qnRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionService(qRepo, qnRepo, rRepo)

	_, appErr := svc.Create(context.Background(), "507f1f77bcf86cd799439011", CreateQuestionInput{
		QuestionKey: "not-a-uuid",
		Schema:      mustQuestionSchemaSingleChoice("非法questionKey"),
	})
	if appErr == nil || appErr.Code != 400 {
		t.Fatalf("expected bad request for non-uuid questionKey, got %+v", appErr)
	}
}

func TestQuestionServiceGetStatsWithVersionAndTimeRange(t *testing.T) {
	qRepo := newFakeQuestionRepo()
	qnRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionService(qRepo, qnRepo, rRepo)

	qRepo.questionsByID["qid-num"] = &domain.QuestionEntity{ID: "qid-num", QuestionKey: uuid.NewString(), OwnerID: "507f1f77bcf86cd799439011", CurrentVersion: 1, CurrentVersionID: "qv-num-v1"}
	qRepo.versionsByID["qv-num-v1"] = &domain.QuestionVersion{ID: "qv-num-v1", QuestionID: "qid-num", Version: 1, Schema: domain.QuestionSchema{Type: domain.QuestionTypeNumber, Title: "年龄"}}

	qnRepo.items["qn-1"] = domain.Questionnaire{ID: "qn-1", Status: domain.QuestionnaireStatusPublished, Questions: []domain.Question{{QuestionID: "qid-num", QuestionVersionID: "qv-num-v1", Type: domain.QuestionTypeNumber}}}
	qnRepo.items["qn-2"] = domain.Questionnaire{ID: "qn-2", Status: domain.QuestionnaireStatusPublished, Questions: []domain.Question{{QuestionID: "qid-num", QuestionVersionID: "qv-num-v1", Type: domain.QuestionTypeNumber}}}

	rRepo.items["qn-1"] = []domain.SurveyResponse{
		{QuestionnaireID: "qn-1", SubmittedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC), Answers: []domain.Answer{{QuestionID: "qid-num", QuestionVersionID: "qv-num-v1", Value: 10.0}}},
		{QuestionnaireID: "qn-1", SubmittedAt: time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC), Answers: []domain.Answer{{QuestionID: "qid-num", QuestionVersionID: "qv-num-v1", Value: 20.0}}},
	}
	rRepo.items["qn-2"] = []domain.SurveyResponse{
		{QuestionnaireID: "qn-2", SubmittedAt: time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC), Answers: []domain.Answer{{QuestionID: "qid-num", QuestionVersionID: "qv-num-v1", Value: 30.0}}},
	}

	from := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 4, 23, 59, 59, 0, time.UTC)
	windowStats, appErr := svc.GetStats(context.Background(), "qid-num", QuestionStatsInput{QuestionVersionID: "qv-num-v1", From: &from, To: &to})
	if appErr != nil {
		t.Fatalf("expected range stats success, got appErr=%v", appErr)
	}
	if windowStats.TotalAnswered != 1 {
		t.Fatalf("expected totalAnswered=1 in range, got %d", windowStats.TotalAnswered)
	}
	if windowStats.AverageValue == nil || *windowStats.AverageValue != 30 {
		t.Fatalf("expected average=30 in range, got %+v", windowStats.AverageValue)
	}

	allStats, appErr := svc.GetStats(context.Background(), "qid-num", QuestionStatsInput{QuestionVersionID: "qv-num-v1"})
	if appErr != nil {
		t.Fatalf("expected overall stats success, got appErr=%v", appErr)
	}
	if allStats.TotalAnswered != 3 {
		t.Fatalf("expected totalAnswered=3, got %d", allStats.TotalAnswered)
	}
	if allStats.AverageValue == nil || *allStats.AverageValue != 20 {
		t.Fatalf("expected average=20, got %+v", allStats.AverageValue)
	}
}
