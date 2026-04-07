package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	"github.com/2018wzh/SimpleSurvey/backend/pkg/apperror"
)

type QuestionnaireService struct {
	questionnaires domain.QuestionnaireRepository
	responses      domain.ResponseRepository
}

func NewQuestionnaireService(questionnaires domain.QuestionnaireRepository, responses domain.ResponseRepository) *QuestionnaireService {
	return &QuestionnaireService{
		questionnaires: questionnaires,
		responses:      responses,
	}
}

func (s *QuestionnaireService) Create(ctx context.Context, creatorID string, input CreateQuestionnaireInput) (string, *apperror.AppError) {
	if strings.TrimSpace(creatorID) == "" {
		return "", apperror.Unauthorized("未授权")
	}
	if err := validateQuestionnaireInput(input); err != nil {
		return "", err
	}

	q := &domain.Questionnaire{
		CreatorID:   creatorID,
		Title:       strings.TrimSpace(input.Title),
		Description: strings.TrimSpace(input.Description),
		Settings:    input.Settings,
		Questions:   input.Questions,
		LogicRules:  input.LogicRules,
		Status:      domain.QuestionnaireStatusDraft,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		IsDeleted:   false,
	}

	if err := s.questionnaires.Create(ctx, q); err != nil {
		return "", apperror.Internal("创建问卷失败")
	}
	return q.ID, nil
}

func (s *QuestionnaireService) ListMine(ctx context.Context, creatorID string, filter domain.QuestionnaireListFilter) ([]QuestionnaireListItem, int64, *apperror.AppError) {
	items, total, err := s.questionnaires.ListByCreator(ctx, creatorID, filter)
	if err != nil {
		return nil, 0, apperror.Internal("查询问卷列表失败")
	}

	result := make([]QuestionnaireListItem, 0, len(items))
	for _, item := range items {
		result = append(result, QuestionnaireListItem{
			ID:        item.ID,
			Title:     item.Title,
			Status:    item.Status,
			CreatedAt: item.CreatedAt,
		})
	}
	return result, total, nil
}

func (s *QuestionnaireService) UpdateStatus(ctx context.Context, creatorID, questionnaireID string, input UpdateQuestionnaireStatusInput) *apperror.AppError {
	if input.Status != domain.QuestionnaireStatusPublished && input.Status != domain.QuestionnaireStatusClosed && input.Status != domain.QuestionnaireStatusDraft {
		return apperror.BadRequest("非法状态，仅支持 draft/published/closed")
	}

	if err := s.questionnaires.UpdateStatus(ctx, questionnaireID, creatorID, input.Status, input.Deadline); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return apperror.NotFound("问卷不存在")
		}
		return apperror.Internal("更新问卷状态失败")
	}
	return nil
}

func (s *QuestionnaireService) GetDetail(ctx context.Context, creatorID, questionnaireID string) (*domain.Questionnaire, *apperror.AppError) {
	questionnaire, err := s.questionnaires.FindByIDAndCreator(ctx, questionnaireID, creatorID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("问卷不存在")
		}
		return nil, apperror.Internal("查询问卷失败")
	}
	return questionnaire, nil
}

func (s *QuestionnaireService) GetStats(ctx context.Context, creatorID, questionnaireID string) (*domain.QuestionnaireStats, *apperror.AppError) {
	questionnaire, err := s.questionnaires.FindByIDAndCreator(ctx, questionnaireID, creatorID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("问卷不存在")
		}
		return nil, apperror.Internal("查询问卷失败")
	}

	responses, err := s.loadAllResponses(ctx, questionnaire.ID)
	if err != nil {
		return nil, apperror.Internal("统计答卷失败")
	}

	stats := s.aggregateStats(*questionnaire, responses)
	return stats, nil
}

func (s *QuestionnaireService) GetResponses(ctx context.Context, creatorID, questionnaireID string, filter domain.ResponseListFilter) ([]domain.SurveyResponse, int64, *apperror.AppError) {
	_, err := s.questionnaires.FindByIDAndCreator(ctx, questionnaireID, creatorID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, 0, apperror.NotFound("问卷不存在")
		}
		return nil, 0, apperror.Internal("查询问卷失败")
	}

	items, total, err := s.responses.ListByQuestionnaire(ctx, questionnaireID, filter)
	if err != nil {
		return nil, 0, apperror.Internal("查询答卷失败")
	}

	if filter.QuestionID != "" || filter.QuestionVersionID != "" {
		for i := range items {
			filtered := make([]domain.Answer, 0, 1)
			for _, ans := range items[i].Answers {
				matchesQuestionID := filter.QuestionID == "" || ans.QuestionID == filter.QuestionID
				matchesQuestionVersionID := filter.QuestionVersionID == "" || ans.QuestionVersionID == filter.QuestionVersionID
				if matchesQuestionID && matchesQuestionVersionID {
					filtered = append(filtered, ans)
				}
			}
			items[i].Answers = filtered
		}
	}

	return items, total, nil
}

func (s *QuestionnaireService) GetSurveyForFill(ctx context.Context, questionnaireID string, requesterID *string) (*domain.Questionnaire, *apperror.AppError) {
	questionnaire, err := s.questionnaires.FindByID(ctx, questionnaireID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("问卷不存在")
		}
		return nil, apperror.Internal("查询问卷失败")
	}

	if questionnaire.Status != domain.QuestionnaireStatusPublished {
		return nil, apperror.Forbidden("问卷未发布或已关闭")
	}
	if questionnaire.Deadline != nil && questionnaire.Deadline.Before(time.Now().UTC()) {
		return nil, apperror.Forbidden("问卷已截止")
	}
	if !questionnaire.Settings.AllowAnonymous && requesterID == nil {
		return nil, apperror.Unauthorized("该问卷需要登录后填写")
	}

	masked := *questionnaire
	masked.CreatorID = ""
	return &masked, nil
}

func (s *QuestionnaireService) SubmitResponse(ctx context.Context, questionnaireID string, userID *string, input SubmitResponseInput, ipAddress string) *apperror.AppError {
	questionnaire, err := s.questionnaires.FindByID(ctx, questionnaireID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return apperror.NotFound("问卷不存在")
		}
		return apperror.Internal("查询问卷失败")
	}

	if questionnaire.Status != domain.QuestionnaireStatusPublished {
		return apperror.Forbidden("问卷未发布或已关闭")
	}
	if questionnaire.Deadline != nil && questionnaire.Deadline.Before(time.Now().UTC()) {
		return apperror.Forbidden("问卷已截止")
	}
	if !questionnaire.Settings.AllowAnonymous && userID == nil {
		return apperror.Unauthorized("该问卷需要登录后填写")
	}
	if !questionnaire.Settings.AllowAnonymous && input.IsAnonymous {
		return apperror.BadRequest("该问卷不支持匿名填写")
	}
	if err := validateAnswers(*questionnaire, input.Answers); err != nil {
		return err
	}

	response := &domain.SurveyResponse{
		QuestionnaireID: questionnaireID,
		IsAnonymous:     input.IsAnonymous,
		UserID:          userID,
		Answers:         input.Answers,
		SubmittedAt:     time.Now().UTC(),
		Statistics: domain.ResponseStatistics{
			CompletionTime: input.Statistics.CompletionTime,
			IPAddress:      ipAddress,
		},
	}
	if input.IsAnonymous {
		response.UserID = nil
	}

	if err := s.responses.Create(ctx, response); err != nil {
		return apperror.Internal("提交答卷失败")
	}
	return nil
}

func (s *QuestionnaireService) loadAllResponses(ctx context.Context, questionnaireID string) ([]domain.SurveyResponse, error) {
	const pageSize = 200
	page := 1
	all := make([]domain.SurveyResponse, 0)
	for {
		items, total, err := s.responses.ListByQuestionnaire(ctx, questionnaireID, domain.ResponseListFilter{Page: page, Limit: pageSize})
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if int64(len(all)) >= total || len(items) == 0 {
			break
		}
		page++
	}
	return all, nil
}

func (s *QuestionnaireService) aggregateStats(questionnaire domain.Questionnaire, responses []domain.SurveyResponse) *domain.QuestionnaireStats {
	questionMap := make(map[string]domain.Question, len(questionnaire.Questions))
	for _, q := range questionnaire.Questions {
		questionMap[questionRefKey(q.QuestionID, q.QuestionVersionID)] = q
	}

	numberSums := map[string]float64{}
	numberCounts := map[string]int{}
	questionStatsMap := map[string]*domain.QuestionStat{}

	for _, q := range questionnaire.Questions {
		key := questionRefKey(q.QuestionID, q.QuestionVersionID)
		schema := questionSchemaFromQuestion(q)
		questionStatsMap[key] = &domain.QuestionStat{
			QuestionID:        q.QuestionID,
			QuestionVersionID: q.QuestionVersionID,
			Type:              schema.Type,
			OptionCounts:      map[string]int{},
			TextAnswers:       []string{},
			TotalAnswered:     0,
		}
	}

	for _, resp := range responses {
		for _, ans := range resp.Answers {
			key := questionRefKey(ans.QuestionID, ans.QuestionVersionID)
			q, ok := questionMap[key]
			if !ok {
				continue
			}
			schema := questionSchemaFromQuestion(q)
			stat := questionStatsMap[key]
			stat.TotalAnswered++
			switch schema.Type {
			case domain.QuestionTypeSingleChoice:
				if opt, ok := ans.Value.(string); ok {
					stat.OptionCounts[opt]++
				}
			case domain.QuestionTypeMultipleChoice:
				if values, ok := toStringSlice(ans.Value); ok {
					for _, opt := range values {
						stat.OptionCounts[opt]++
					}
				}
			case domain.QuestionTypeText:
				if text, ok := ans.Value.(string); ok {
					stat.TextAnswers = append(stat.TextAnswers, text)
				}
			case domain.QuestionTypeNumber:
				if number, ok := toFloat64(ans.Value); ok {
					numberSums[key] += number
					numberCounts[key]++
				}
			}
		}
	}

	questionStats := make([]domain.QuestionStat, 0, len(questionnaire.Questions))
	for _, q := range questionnaire.Questions {
		key := questionRefKey(q.QuestionID, q.QuestionVersionID)
		schema := questionSchemaFromQuestion(q)
		stat := questionStatsMap[key]
		if schema.Type != domain.QuestionTypeSingleChoice && schema.Type != domain.QuestionTypeMultipleChoice {
			stat.OptionCounts = nil
		}
		if schema.Type != domain.QuestionTypeText {
			stat.TextAnswers = nil
		}
		if schema.Type == domain.QuestionTypeNumber && numberCounts[key] > 0 {
			avg := numberSums[key] / float64(numberCounts[key])
			stat.AverageValue = &avg
		}
		questionStats = append(questionStats, *stat)
	}

	sort.SliceStable(questionStats, func(i, j int) bool {
		left := questionStats[i].QuestionID + "::" + questionStats[i].QuestionVersionID
		right := questionStats[j].QuestionID + "::" + questionStats[j].QuestionVersionID
		return left < right
	})

	return &domain.QuestionnaireStats{
		TotalResponses: int64(len(responses)),
		QuestionStats:  questionStats,
	}
}
