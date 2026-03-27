package service

import (
	"context"
	"testing"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
)

type fakeQuestionnaireRepository struct {
	items map[string]domain.Questionnaire
}

func newFakeQuestionnaireRepo() *fakeQuestionnaireRepository {
	return &fakeQuestionnaireRepository{items: map[string]domain.Questionnaire{}}
}

func (f *fakeQuestionnaireRepository) Create(_ context.Context, questionnaire *domain.Questionnaire) error {
	if questionnaire.ID == "" {
		questionnaire.ID = questionnaire.Title + "-id"
	}
	f.items[questionnaire.ID] = *questionnaire
	return nil
}

func (f *fakeQuestionnaireRepository) FindByID(_ context.Context, id string) (*domain.Questionnaire, error) {
	item, ok := f.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := item
	return &copy, nil
}

func (f *fakeQuestionnaireRepository) FindByIDAndCreator(_ context.Context, id, creatorID string) (*domain.Questionnaire, error) {
	item, ok := f.items[id]
	if !ok || item.CreatorID != creatorID {
		return nil, domain.ErrNotFound
	}
	copy := item
	return &copy, nil
}

func (f *fakeQuestionnaireRepository) ListByCreator(_ context.Context, creatorID string, filter domain.QuestionnaireListFilter) ([]domain.Questionnaire, int64, error) {
	list := make([]domain.Questionnaire, 0)
	for _, item := range f.items {
		if item.CreatorID == creatorID {
			list = append(list, item)
		}
	}
	return list, int64(len(list)), nil
}

func (f *fakeQuestionnaireRepository) ListAll(_ context.Context, filter domain.QuestionnaireAdminListFilter) ([]domain.Questionnaire, int64, error) {
	list := make([]domain.Questionnaire, 0)
	for _, item := range f.items {
		if filter.CreatorID != "" && item.CreatorID != filter.CreatorID {
			continue
		}
		if filter.Status != "" && string(item.Status) != filter.Status {
			continue
		}
		list = append(list, item)
	}
	return list, int64(len(list)), nil
}

func (f *fakeQuestionnaireRepository) UpdateStatus(_ context.Context, id, creatorID string, status domain.QuestionnaireStatus, deadline *time.Time) error {
	item, ok := f.items[id]
	if !ok || item.CreatorID != creatorID {
		return domain.ErrNotFound
	}
	item.Status = status
	item.Deadline = deadline
	item.UpdatedAt = time.Now().UTC()
	f.items[id] = item
	return nil
}

func (f *fakeQuestionnaireRepository) UpdateStatusByAdmin(_ context.Context, id string, status domain.QuestionnaireStatus, deadline *time.Time) error {
	item, ok := f.items[id]
	if !ok {
		return domain.ErrNotFound
	}
	item.Status = status
	item.Deadline = deadline
	item.UpdatedAt = time.Now().UTC()
	f.items[id] = item
	return nil
}

type fakeResponseRepository struct {
	items map[string][]domain.SurveyResponse
}

func newFakeResponseRepo() *fakeResponseRepository {
	return &fakeResponseRepository{items: map[string][]domain.SurveyResponse{}}
}

func (f *fakeResponseRepository) Create(_ context.Context, response *domain.SurveyResponse) error {
	if response.ID == "" {
		response.ID = "resp-" + time.Now().Format("150405.000")
	}
	f.items[response.QuestionnaireID] = append(f.items[response.QuestionnaireID], *response)
	return nil
}

func (f *fakeResponseRepository) ListByQuestionnaire(_ context.Context, questionnaireID string, filter domain.ResponseListFilter) ([]domain.SurveyResponse, int64, error) {
	list := f.items[questionnaireID]
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	start := (filter.Page - 1) * filter.Limit
	if start >= len(list) {
		return []domain.SurveyResponse{}, int64(len(list)), nil
	}
	end := start + filter.Limit
	if end > len(list) {
		end = len(list)
	}

	out := make([]domain.SurveyResponse, 0, end-start)
	for _, item := range list[start:end] {
		if filter.QuestionID != "" {
			matched := false
			for _, ans := range item.Answers {
				if ans.QuestionID == filter.QuestionID {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, item)
	}
	return out, int64(len(list)), nil
}

func (f *fakeResponseRepository) CountByQuestionnaire(_ context.Context, questionnaireID string) (int64, error) {
	return int64(len(f.items[questionnaireID])), nil
}

func TestCreateQuestionnaireSuccess(t *testing.T) {
	qRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionnaireService(qRepo, rRepo)

	id, appErr := svc.Create(context.Background(), "u1", CreateQuestionnaireInput{
		Title:       "满意度调查",
		Description: "desc",
		Settings: domain.QuestionnaireSettings{
			AllowAnonymous: true,
		},
		Questions: []domain.Question{
			{
				QuestionID: "q1",
				Type:       domain.QuestionTypeSingleChoice,
				Title:      "是否满意",
				IsRequired: true,
				Options:    []domain.QuestionOption{{OptionID: "o1", Text: "是"}, {OptionID: "o2", Text: "否"}},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("unexpected app error: %v", appErr)
	}
	if id == "" {
		t.Fatal("expected questionnaire id")
	}
	stored, _ := qRepo.FindByID(context.Background(), id)
	if stored.Status != domain.QuestionnaireStatusDraft {
		t.Fatalf("expected draft status, got %s", stored.Status)
	}
}

func TestCreateQuestionnaireFailWhenLogicRuleInvalid(t *testing.T) {
	qRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionnaireService(qRepo, rRepo)

	_, appErr := svc.Create(context.Background(), "u1", CreateQuestionnaireInput{
		Title:    "调查",
		Settings: domain.QuestionnaireSettings{AllowAnonymous: true},
		Questions: []domain.Question{{
			QuestionID: "q1",
			Type:       domain.QuestionTypeSingleChoice,
			Title:      "题1",
			IsRequired: true,
			Options:    []domain.QuestionOption{{OptionID: "a", Text: "A"}, {OptionID: "b", Text: "B"}},
		}},
		LogicRules: []domain.LogicRule{{
			ConditionQuestionID: "q1",
			Operator:            domain.LogicOperatorEquals,
			ConditionValue:      "a",
			Action:              domain.LogicActionJumpTo,
			ActionDetails:       map[string]interface{}{"targetQuestionId": "q_not_found"},
		}},
	})
	if appErr == nil {
		t.Fatal("expected validation error for invalid targetQuestionId")
	}
}

func TestSubmitResponseValidationErrorForMultiChoiceLimit(t *testing.T) {
	qRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionnaireService(qRepo, rRepo)

	qRepo.items["q123"] = domain.Questionnaire{
		ID:        "q123",
		CreatorID: "owner",
		Status:    domain.QuestionnaireStatusPublished,
		Settings:  domain.QuestionnaireSettings{AllowAnonymous: true},
		Questions: []domain.Question{{
			QuestionID: "q1",
			Type:       domain.QuestionTypeMultipleChoice,
			Title:      "选择功能",
			IsRequired: true,
			Options:    []domain.QuestionOption{{OptionID: "a", Text: "A"}, {OptionID: "b", Text: "B"}, {OptionID: "c", Text: "C"}},
			Validation: domain.QuestionValidation{MinSelect: intPtr(1), MaxSelect: intPtr(2)},
		}},
	}

	err := svc.SubmitResponse(context.Background(), "q123", nil, SubmitResponseInput{
		IsAnonymous: true,
		Answers:     []domain.Answer{{QuestionID: "q1", Value: []interface{}{"a", "b", "c"}}},
	}, "127.0.0.1")
	if err == nil {
		t.Fatal("expected maxSelect validation error")
	}
}

func TestSubmitResponseSuccess(t *testing.T) {
	qRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionnaireService(qRepo, rRepo)

	qRepo.items["q321"] = domain.Questionnaire{
		ID:        "q321",
		CreatorID: "owner",
		Status:    domain.QuestionnaireStatusPublished,
		Settings:  domain.QuestionnaireSettings{AllowAnonymous: true},
		Questions: []domain.Question{{
			QuestionID: "q1",
			Type:       domain.QuestionTypeSingleChoice,
			Title:      "满意吗",
			IsRequired: true,
			Options:    []domain.QuestionOption{{OptionID: "yes", Text: "是"}, {OptionID: "no", Text: "否"}},
		}},
	}

	err := svc.SubmitResponse(context.Background(), "q321", nil, SubmitResponseInput{
		IsAnonymous: true,
		Answers:     []domain.Answer{{QuestionID: "q1", Value: "yes"}},
	}, "127.0.0.1")
	if err != nil {
		t.Fatalf("expected submit success, got err: %v", err)
	}
	if len(rRepo.items["q321"]) != 1 {
		t.Fatalf("expected one response, got %d", len(rRepo.items["q321"]))
	}
}

func TestGetStatsAggregatesOptionAndAverage(t *testing.T) {
	qRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionnaireService(qRepo, rRepo)

	qRepo.items["q999"] = domain.Questionnaire{
		ID:        "q999",
		CreatorID: "owner",
		Status:    domain.QuestionnaireStatusPublished,
		Questions: []domain.Question{
			{QuestionID: "q1", Type: domain.QuestionTypeSingleChoice, Title: "Q1", Options: []domain.QuestionOption{{OptionID: "a", Text: "A"}, {OptionID: "b", Text: "B"}}},
			{QuestionID: "q2", Type: domain.QuestionTypeNumber, Title: "年龄", Validation: domain.QuestionValidation{NumberType: "integer"}},
		},
	}

	_ = rRepo.Create(context.Background(), &domain.SurveyResponse{QuestionnaireID: "q999", Answers: []domain.Answer{{QuestionID: "q1", Value: "a"}, {QuestionID: "q2", Value: 20.0}}})
	_ = rRepo.Create(context.Background(), &domain.SurveyResponse{QuestionnaireID: "q999", Answers: []domain.Answer{{QuestionID: "q1", Value: "a"}, {QuestionID: "q2", Value: 30.0}}})
	_ = rRepo.Create(context.Background(), &domain.SurveyResponse{QuestionnaireID: "q999", Answers: []domain.Answer{{QuestionID: "q1", Value: "b"}, {QuestionID: "q2", Value: 40.0}}})

	stats, err := svc.GetStats(context.Background(), "owner", "q999")
	if err != nil {
		t.Fatalf("expected stats success, got err: %v", err)
	}
	if stats.TotalResponses != 3 {
		t.Fatalf("expected totalResponses=3, got %d", stats.TotalResponses)
	}

	statMap := map[string]domain.QuestionStat{}
	for _, s := range stats.QuestionStats {
		statMap[s.QuestionID] = s
	}
	if statMap["q1"].OptionCounts["a"] != 2 || statMap["q1"].OptionCounts["b"] != 1 {
		t.Fatalf("unexpected single choice counts: %+v", statMap["q1"].OptionCounts)
	}
	if statMap["q2"].AverageValue == nil || *statMap["q2"].AverageValue != 30 {
		t.Fatalf("unexpected average value: %+v", statMap["q2"].AverageValue)
	}
}

func TestGetSurveyRequiresLoginWhenNotAllowAnonymous(t *testing.T) {
	qRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionnaireService(qRepo, rRepo)

	qRepo.items["q-auth"] = domain.Questionnaire{
		ID:        "q-auth",
		CreatorID: "owner",
		Status:    domain.QuestionnaireStatusPublished,
		Settings: domain.QuestionnaireSettings{
			AllowAnonymous: false,
		},
		Questions: []domain.Question{{
			QuestionID: "q1",
			Type:       domain.QuestionTypeText,
			Title:      "建议",
		}},
	}

	_, appErr := svc.GetSurveyForFill(context.Background(), "q-auth", nil)
	if appErr == nil {
		t.Fatal("expected unauthorized error when anonymous is not allowed")
	}
}

func intPtr(v int) *int { return &v }
