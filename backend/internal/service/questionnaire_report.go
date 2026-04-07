package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	"github.com/2018wzh/SimpleSurvey/backend/pkg/apperror"
)

func (s *QuestionnaireService) BuildCrossTab(ctx context.Context, creatorID, questionnaireID string, input CrossTabInput) (*domain.CrossTabReport, *apperror.AppError) {
	rowQuestionID := strings.TrimSpace(input.RowQuestionID)
	colQuestionID := strings.TrimSpace(input.ColQuestionID)
	completionStatus := strings.ToLower(strings.TrimSpace(input.Filters.CompletionStatus))
	if rowQuestionID == "" || colQuestionID == "" {
		return nil, apperror.BadRequest("rowQuestionId和colQuestionId不能为空")
	}
	if completionStatus != "" && completionStatus != "completed" {
		return nil, apperror.BadRequest("filters.completionStatus仅支持completed")
	}
	if input.Filters.DateRange.Start != nil && input.Filters.DateRange.End != nil && input.Filters.DateRange.Start.After(*input.Filters.DateRange.End) {
		return nil, apperror.BadRequest("filters.dateRange.start不能晚于end")
	}

	questionnaire, err := s.questionnaires.FindByIDAndCreator(ctx, questionnaireID, creatorID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("问卷不存在")
		}
		return nil, apperror.Internal("生成交叉报表失败")
	}

	questionExists := map[string]bool{}
	for _, q := range questionnaire.Questions {
		questionExists[q.QuestionID] = true
	}
	if !questionExists[rowQuestionID] || !questionExists[colQuestionID] {
		return nil, apperror.BadRequest("rowQuestionId或colQuestionId不在当前问卷中")
	}

	responses, err := s.loadAllResponses(ctx, questionnaire.ID)
	if err != nil {
		return nil, apperror.Internal("生成交叉报表失败")
	}

	matrixCounter := map[string]map[string]int{}
	totalSample := 0
	for _, resp := range responses {
		if input.Filters.DateRange.Start != nil && resp.SubmittedAt.Before(*input.Filters.DateRange.Start) {
			continue
		}
		if input.Filters.DateRange.End != nil && resp.SubmittedAt.After(*input.Filters.DateRange.End) {
			continue
		}
		rowVal, hasRow := findAnswerBucket(resp.Answers, rowQuestionID)
		colVal, hasCol := findAnswerBucket(resp.Answers, colQuestionID)
		if !hasRow || !hasCol {
			continue
		}
		if _, ok := matrixCounter[rowVal]; !ok {
			matrixCounter[rowVal] = map[string]int{}
		}
		matrixCounter[rowVal][colVal]++
		totalSample++
	}

	rows := make([]string, 0, len(matrixCounter))
	for row := range matrixCounter {
		rows = append(rows, row)
	}
	sort.Strings(rows)

	matrix := make([]domain.CrossTabCell, 0)
	for _, row := range rows {
		cols := make([]string, 0, len(matrixCounter[row]))
		for col := range matrixCounter[row] {
			cols = append(cols, col)
		}
		sort.Strings(cols)
		for _, col := range cols {
			count := matrixCounter[row][col]
			percentage := 0.0
			if totalSample > 0 {
				percentage = float64(count) / float64(totalSample)
			}
			matrix = append(matrix, domain.CrossTabCell{Row: row, Col: col, Count: count, Percentage: percentage})
		}
	}

	return &domain.CrossTabReport{
		RowQuestionID: rowQuestionID,
		ColQuestionID: colQuestionID,
		TotalSample:   totalSample,
		Matrix:        matrix,
	}, nil
}

func findAnswerBucket(answers []domain.Answer, questionID string) (string, bool) {
	for _, ans := range answers {
		if ans.QuestionID != questionID {
			continue
		}
		return answerBucket(ans.Value), true
	}
	return "", false
}

func answerBucket(value interface{}) string {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return "(empty)"
		}
		return v
	case []string:
		copied := append([]string{}, v...)
		sort.Strings(copied)
		return strings.Join(copied, "|")
	case []interface{}:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, fmt.Sprint(item))
		}
		sort.Strings(parts)
		return strings.Join(parts, "|")
	default:
		return fmt.Sprint(v)
	}
}
