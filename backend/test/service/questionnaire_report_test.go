package service

import (
	"context"
	"testing"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	. "github.com/2018wzh/SimpleSurvey/backend/internal/service"
)

func TestBuildCrossTab(t *testing.T) {
	qRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionnaireService(qRepo, rRepo)

	qRepo.items["qn-100"] = domain.Questionnaire{
		ID:        "qn-100",
		CreatorID: "u1",
		Questions: []domain.Question{{QuestionID: "q1", QuestionVersionID: "q1-v1", Type: domain.QuestionTypeSingleChoice}, {QuestionID: "q2", QuestionVersionID: "q2-v1", Type: domain.QuestionTypeSingleChoice}},
	}
	rRepo.items["qn-100"] = []domain.SurveyResponse{
		{QuestionnaireID: "qn-100", Answers: []domain.Answer{{QuestionID: "q1", QuestionVersionID: "q1-v1", Value: "A"}, {QuestionID: "q2", QuestionVersionID: "q2-v1", Value: "X"}}},
		{QuestionnaireID: "qn-100", Answers: []domain.Answer{{QuestionID: "q1", QuestionVersionID: "q1-v1", Value: "A"}, {QuestionID: "q2", QuestionVersionID: "q2-v1", Value: "Y"}}},
		{QuestionnaireID: "qn-100", Answers: []domain.Answer{{QuestionID: "q1", QuestionVersionID: "q1-v1", Value: "B"}, {QuestionID: "q2", QuestionVersionID: "q2-v1", Value: "X"}}},
	}

	report, appErr := svc.BuildCrossTab(context.Background(), "u1", "qn-100", CrossTabInput{RowQuestionID: "q1", ColQuestionID: "q2"})
	if appErr != nil {
		t.Fatalf("expected crosstab success, got appErr=%v", appErr)
	}
	if report.TotalSample != 3 {
		t.Fatalf("expected total sample 3, got %d", report.TotalSample)
	}
	if len(report.Matrix) != 3 {
		t.Fatalf("expected 3 crosstab cells, got %d", len(report.Matrix))
	}
}

func TestBuildCrossTabNormalizesMultiChoiceBuckets(t *testing.T) {
	qRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionnaireService(qRepo, rRepo)

	qRepo.items["qn-bucket"] = domain.Questionnaire{
		ID:        "qn-bucket",
		CreatorID: "u1",
		Questions: []domain.Question{{QuestionID: "q1", QuestionVersionID: "q1-v1", Type: domain.QuestionTypeMultipleChoice}, {QuestionID: "q2", QuestionVersionID: "q2-v1", Type: domain.QuestionTypeSingleChoice}},
	}
	rRepo.items["qn-bucket"] = []domain.SurveyResponse{
		{QuestionnaireID: "qn-bucket", Answers: []domain.Answer{{QuestionID: "q1", QuestionVersionID: "q1-v1", Value: []interface{}{"b", "a"}}, {QuestionID: "q2", QuestionVersionID: "q2-v1", Value: "X"}}},
		{QuestionnaireID: "qn-bucket", Answers: []domain.Answer{{QuestionID: "q1", QuestionVersionID: "q1-v1", Value: []interface{}{"a", "b"}}, {QuestionID: "q2", QuestionVersionID: "q2-v1", Value: "X"}}},
		{QuestionnaireID: "qn-bucket", Answers: []domain.Answer{{QuestionID: "q1", QuestionVersionID: "q1-v1", Value: []interface{}{"a"}}}},
	}

	report, appErr := svc.BuildCrossTab(context.Background(), "u1", "qn-bucket", CrossTabInput{RowQuestionID: "q1", ColQuestionID: "q2"})
	if appErr != nil {
		t.Fatalf("expected crosstab success, got appErr=%v", appErr)
	}
	if report.TotalSample != 2 {
		t.Fatalf("expected total sample 2 (skip incomplete), got %d", report.TotalSample)
	}
	if len(report.Matrix) != 1 {
		t.Fatalf("expected one matrix cell, got %d", len(report.Matrix))
	}
	if report.Matrix[0].Row != "a|b" || report.Matrix[0].Col != "X" || report.Matrix[0].Count != 2 {
		t.Fatalf("unexpected normalized bucket cell: %+v", report.Matrix[0])
	}
	if report.Matrix[0].Percentage != 1 {
		t.Fatalf("expected percentage=1, got %f", report.Matrix[0].Percentage)
	}
}

func TestBuildCrossTabRejectsQuestionOutsideQuestionnaire(t *testing.T) {
	qRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionnaireService(qRepo, rRepo)

	qRepo.items["qn-invalid"] = domain.Questionnaire{
		ID:        "qn-invalid",
		CreatorID: "u1",
		Questions: []domain.Question{{QuestionID: "q1", QuestionVersionID: "q1-v1", Type: domain.QuestionTypeSingleChoice}},
	}

	_, appErr := svc.BuildCrossTab(context.Background(), "u1", "qn-invalid", CrossTabInput{RowQuestionID: "q1", ColQuestionID: "q-not-exist"})
	if appErr == nil || appErr.Code != 400 {
		t.Fatalf("expected bad request when row/col question not in questionnaire, got %+v", appErr)
	}
}

func TestBuildCrossTabWithDateRangeFilter(t *testing.T) {
	qRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionnaireService(qRepo, rRepo)

	qRepo.items["qn-filter"] = domain.Questionnaire{
		ID:        "qn-filter",
		CreatorID: "u1",
		Questions: []domain.Question{{QuestionID: "q1", QuestionVersionID: "q1-v1", Type: domain.QuestionTypeSingleChoice}, {QuestionID: "q2", QuestionVersionID: "q2-v1", Type: domain.QuestionTypeSingleChoice}},
	}
	rRepo.items["qn-filter"] = []domain.SurveyResponse{
		{QuestionnaireID: "qn-filter", SubmittedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC), Answers: []domain.Answer{{QuestionID: "q1", QuestionVersionID: "q1-v1", Value: "A"}, {QuestionID: "q2", QuestionVersionID: "q2-v1", Value: "X"}}},
		{QuestionnaireID: "qn-filter", SubmittedAt: time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC), Answers: []domain.Answer{{QuestionID: "q1", QuestionVersionID: "q1-v1", Value: "A"}, {QuestionID: "q2", QuestionVersionID: "q2-v1", Value: "Y"}}},
	}

	start := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)
	report, appErr := svc.BuildCrossTab(context.Background(), "u1", "qn-filter", CrossTabInput{
		RowQuestionID: "q1",
		ColQuestionID: "q2",
		Filters: CrossTabFilters{
			DateRange: CrossTabDateRange{Start: &start, End: &end},
		},
	})
	if appErr != nil {
		t.Fatalf("expected crosstab with date filter success, got appErr=%v", appErr)
	}
	if report.TotalSample != 1 {
		t.Fatalf("expected total sample=1 after date filtering, got %d", report.TotalSample)
	}
	if len(report.Matrix) != 1 || report.Matrix[0].Col != "Y" {
		t.Fatalf("unexpected filtered matrix: %+v", report.Matrix)
	}
}

func TestBuildCrossTabRejectsUnsupportedCompletionStatus(t *testing.T) {
	qRepo := newFakeQuestionnaireRepo()
	rRepo := newFakeResponseRepo()
	svc := NewQuestionnaireService(qRepo, rRepo)

	qRepo.items["qn-completion"] = domain.Questionnaire{
		ID:        "qn-completion",
		CreatorID: "u1",
		Questions: []domain.Question{{QuestionID: "q1", QuestionVersionID: "q1-v1", Type: domain.QuestionTypeSingleChoice}, {QuestionID: "q2", QuestionVersionID: "q2-v1", Type: domain.QuestionTypeSingleChoice}},
	}

	_, appErr := svc.BuildCrossTab(context.Background(), "u1", "qn-completion", CrossTabInput{
		RowQuestionID: "q1",
		ColQuestionID: "q2",
		Filters:       CrossTabFilters{CompletionStatus: "in_progress"},
	})
	if appErr == nil || appErr.Code != 400 {
		t.Fatalf("expected 400 for unsupported completionStatus, got %+v", appErr)
	}
}
