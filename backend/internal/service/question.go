package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	"github.com/2018wzh/SimpleSurvey/backend/pkg/apperror"
	"github.com/google/uuid"
)

type QuestionService struct {
	questions      domain.QuestionRepository
	questionnaires domain.QuestionnaireRepository
	responses      domain.ResponseRepository
}

func NewQuestionService(questions domain.QuestionRepository, questionnaires domain.QuestionnaireRepository, responses domain.ResponseRepository) *QuestionService {
	return &QuestionService{questions: questions, questionnaires: questionnaires, responses: responses}
}

func (s *QuestionService) Create(ctx context.Context, ownerID string, input CreateQuestionInput) (*CreateQuestionResult, *apperror.AppError) {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return nil, apperror.Unauthorized("未授权")
	}
	if err := validateQuestionSchema(input.Schema); err != nil {
		return nil, err
	}
	questionKey := strings.TrimSpace(input.QuestionKey)
	if questionKey == "" {
		return nil, apperror.BadRequest("questionKey不能为空")
	}
	if _, err := uuid.Parse(questionKey); err != nil {
		return nil, apperror.BadRequest("questionKey必须是合法UUID")
	}

	if _, err := s.questions.FindByQuestionKey(ctx, questionKey); err == nil {
		return nil, apperror.Conflict("questionKey已存在")
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.Internal("创建题目失败")
	}

	now := time.Now().UTC()
	question := &domain.QuestionEntity{
		QuestionKey:    questionKey,
		OwnerID:        ownerID,
		CurrentVersion: 1,
		Tags:           normalizeTags(input.Tags),
		CreatedAt:      now,
		UpdatedAt:      now,
		IsArchived:     false,
	}
	version := &domain.QuestionVersion{
		Version:    1,
		ChangeType: domain.QuestionVersionChangeTypeCreate,
		Schema:     input.Schema,
		CreatedBy:  ownerID,
		CreatedAt:  now,
	}

	if err := s.questions.Create(ctx, question, version); err != nil {
		if errors.Is(err, domain.ErrDuplicate) {
			return nil, apperror.Conflict("questionKey已存在")
		}
		return nil, apperror.Internal("创建题目失败")
	}

	return &CreateQuestionResult{ID: question.ID, Version: 1, VersionID: version.ID}, nil
}

func (s *QuestionService) CreateVersion(ctx context.Context, ownerID, questionID string, input CreateQuestionVersionInput) (*CreateQuestionResult, *apperror.AppError) {
	ownerID = strings.TrimSpace(ownerID)
	questionID = strings.TrimSpace(questionID)
	if ownerID == "" {
		return nil, apperror.Unauthorized("未授权")
	}
	if questionID == "" {
		return nil, apperror.BadRequest("questionId不能为空")
	}
	if err := validateQuestionSchema(input.Schema); err != nil {
		return nil, err
	}

	question, err := s.questions.FindByIDAndOwner(ctx, questionID, ownerID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("题目不存在")
		}
		return nil, apperror.Internal("创建题目版本失败")
	}

	changeType := input.ChangeType
	if changeType == "" {
		changeType = domain.QuestionVersionChangeTypeEdit
	}

	parentVersionID := strings.TrimSpace(input.BaseVersionID)
	if parentVersionID == "" {
		parentVersionID = question.CurrentVersionID
	}
	parentVersion, err := s.questions.FindVersionByID(ctx, parentVersionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.BadRequest("baseVersionId不存在")
		}
		return nil, apperror.Internal("创建题目版本失败")
	}
	if parentVersion.QuestionID != question.ID {
		return nil, apperror.BadRequest("baseVersionId不属于当前题目")
	}

	now := time.Now().UTC()
	version := &domain.QuestionVersion{
		QuestionID:      question.ID,
		Version:         question.CurrentVersion + 1,
		ParentVersionID: &parentVersion.ID,
		ChangeType:      changeType,
		Schema:          input.Schema,
		CreatedBy:       ownerID,
		CreatedAt:       now,
		Note:            strings.TrimSpace(input.Note),
	}
	question.UpdatedAt = now

	if err := s.questions.CreateVersion(ctx, question, version); err != nil {
		if errors.Is(err, domain.ErrDuplicate) {
			return nil, apperror.Conflict("版本冲突，请重试")
		}
		return nil, apperror.Internal("创建题目版本失败")
	}

	return &CreateQuestionResult{ID: question.ID, Version: version.Version, VersionID: version.ID}, nil
}

func (s *QuestionService) ListVersions(ctx context.Context, ownerID, questionID string) ([]domain.QuestionVersion, *apperror.AppError) {
	ownerID = strings.TrimSpace(ownerID)
	questionID = strings.TrimSpace(questionID)
	if ownerID == "" {
		return nil, apperror.Unauthorized("未授权")
	}
	if _, err := s.questions.FindByIDAndOwner(ctx, questionID, ownerID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("题目不存在")
		}
		return nil, apperror.Internal("查询题目版本失败")
	}

	versions, err := s.questions.ListVersions(ctx, questionID)
	if err != nil {
		return nil, apperror.Internal("查询题目版本失败")
	}
	return versions, nil
}

func (s *QuestionService) RestoreVersion(ctx context.Context, ownerID, questionID string, input RestoreQuestionVersionInput) (*CreateQuestionResult, *apperror.AppError) {
	ownerID = strings.TrimSpace(ownerID)
	questionID = strings.TrimSpace(questionID)
	fromVersionID := strings.TrimSpace(input.FromVersionID)
	if ownerID == "" {
		return nil, apperror.Unauthorized("未授权")
	}
	if fromVersionID == "" {
		return nil, apperror.BadRequest("fromVersionId不能为空")
	}

	question, err := s.questions.FindByIDAndOwner(ctx, questionID, ownerID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("题目不存在")
		}
		return nil, apperror.Internal("恢复题目版本失败")
	}

	fromVersion, err := s.questions.FindVersionByID(ctx, fromVersionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.BadRequest("fromVersionId不存在")
		}
		return nil, apperror.Internal("恢复题目版本失败")
	}
	if fromVersion.QuestionID != question.ID {
		return nil, apperror.BadRequest("fromVersionId不属于当前题目")
	}

	now := time.Now().UTC()
	version := &domain.QuestionVersion{
		QuestionID:      question.ID,
		Version:         question.CurrentVersion + 1,
		ParentVersionID: &fromVersion.ID,
		ChangeType:      domain.QuestionVersionChangeTypeRestore,
		Schema:          fromVersion.Schema,
		CreatedBy:       ownerID,
		CreatedAt:       now,
		Note:            strings.TrimSpace(input.Note),
	}
	question.UpdatedAt = now

	if err := s.questions.CreateVersion(ctx, question, version); err != nil {
		if errors.Is(err, domain.ErrDuplicate) {
			return nil, apperror.Conflict("版本冲突，请重试")
		}
		return nil, apperror.Internal("恢复题目版本失败")
	}

	return &CreateQuestionResult{ID: question.ID, Version: version.Version, VersionID: version.ID}, nil
}

func (s *QuestionService) GetUsages(ctx context.Context, questionID, questionVersionID, status string) ([]domain.QuestionUsage, *apperror.AppError) {
	questionID = strings.TrimSpace(questionID)
	questionVersionID = strings.TrimSpace(questionVersionID)
	status = strings.TrimSpace(status)

	if _, err := s.questions.FindByID(ctx, questionID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("题目不存在")
		}
		return nil, apperror.Internal("查询题目使用情况失败")
	}

	questionnaires, err := s.loadAllQuestionnaires(ctx)
	if err != nil {
		return nil, apperror.Internal("查询题目使用情况失败")
	}

	usages := make([]domain.QuestionUsage, 0)
	for _, q := range questionnaires {
		if status != "" && string(q.Status) != status {
			continue
		}
		for _, ref := range q.Questions {
			if ref.QuestionID != questionID {
				continue
			}
			if questionVersionID != "" && ref.QuestionVersionID != questionVersionID {
				continue
			}
			usages = append(usages, domain.QuestionUsage{
				QuestionnaireID:    q.ID,
				QuestionnaireTitle: q.Title,
				Status:             q.Status,
				QuestionVersionID:  ref.QuestionVersionID,
			})
		}
	}

	sort.SliceStable(usages, func(i, j int) bool {
		if usages[i].QuestionnaireID == usages[j].QuestionnaireID {
			return usages[i].QuestionVersionID < usages[j].QuestionVersionID
		}
		return usages[i].QuestionnaireID < usages[j].QuestionnaireID
	})
	return usages, nil
}

func (s *QuestionService) GetStats(ctx context.Context, questionID string, input QuestionStatsInput) (*domain.QuestionCrossStats, *apperror.AppError) {
	questionID = strings.TrimSpace(questionID)
	questionVersionID := strings.TrimSpace(input.QuestionVersionID)

	question, err := s.questions.FindByID(ctx, questionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("题目不存在")
		}
		return nil, apperror.Internal("查询题目统计失败")
	}

	var targetVersion *domain.QuestionVersion
	if questionVersionID != "" {
		v, err := s.questions.FindVersionByID(ctx, questionVersionID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, apperror.BadRequest("questionVersionId不存在")
			}
			return nil, apperror.Internal("查询题目统计失败")
		}
		if v.QuestionID != question.ID {
			return nil, apperror.BadRequest("questionVersionId不属于该题目")
		}
		targetVersion = v
	} else {
		v, err := s.questions.FindVersionByID(ctx, question.CurrentVersionID)
		if err == nil {
			targetVersion = v
		}
	}

	qType := domain.QuestionTypeText
	if targetVersion != nil {
		qType = targetVersion.Schema.Type
	}

	questionnaires, err := s.loadAllQuestionnaires(ctx)
	if err != nil {
		return nil, apperror.Internal("查询题目统计失败")
	}

	usageQuestionnaireIDs := map[string]struct{}{}
	for _, q := range questionnaires {
		for _, ref := range q.Questions {
			if ref.QuestionID != questionID {
				continue
			}
			if questionVersionID != "" && ref.QuestionVersionID != questionVersionID {
				continue
			}
			usageQuestionnaireIDs[q.ID] = struct{}{}
		}
	}

	result := &domain.QuestionCrossStats{
		QuestionID:        questionID,
		QuestionVersionID: questionVersionID,
		Type:              qType,
		OptionCounts:      map[string]int{},
		TextAnswers:       []string{},
	}

	for questionnaireID := range usageQuestionnaireIDs {
		responses, err := s.loadAllResponses(ctx, questionnaireID)
		if err != nil {
			return nil, apperror.Internal("查询题目统计失败")
		}
		for _, resp := range responses {
			if input.From != nil && resp.SubmittedAt.Before(*input.From) {
				continue
			}
			if input.To != nil && resp.SubmittedAt.After(*input.To) {
				continue
			}
			for _, answer := range resp.Answers {
				if answer.QuestionID != questionID {
					continue
				}
				if questionVersionID != "" && answer.QuestionVersionID != questionVersionID {
					continue
				}
				result.TotalAnswered++
				switch qType {
				case domain.QuestionTypeSingleChoice:
					if opt, ok := answer.Value.(string); ok {
						result.OptionCounts[opt]++
					}
				case domain.QuestionTypeMultipleChoice:
					if values, ok := toStringSlice(answer.Value); ok {
						for _, opt := range values {
							result.OptionCounts[opt]++
						}
					}
				case domain.QuestionTypeNumber:
					if value, ok := toFloat64(answer.Value); ok {
						if result.AverageValue == nil {
							base := 0.0
							result.AverageValue = &base
						}
						*result.AverageValue += value
					}
				default:
					if txt, ok := answer.Value.(string); ok {
						result.TextAnswers = append(result.TextAnswers, txt)
					}
				}
			}
		}
	}

	if qType == domain.QuestionTypeNumber {
		if result.TotalAnswered > 0 && result.AverageValue != nil {
			avg := *result.AverageValue / float64(result.TotalAnswered)
			result.AverageValue = &avg
		}
		result.OptionCounts = nil
		result.TextAnswers = nil
	} else if qType == domain.QuestionTypeText {
		result.OptionCounts = nil
		result.AverageValue = nil
	} else {
		result.TextAnswers = nil
		result.AverageValue = nil
	}

	return result, nil
}

func (s *QuestionService) loadAllQuestionnaires(ctx context.Context) ([]domain.Questionnaire, error) {
	const pageSize = 100
	page := 1
	all := make([]domain.Questionnaire, 0)
	for {
		items, total, err := s.questionnaires.ListAll(ctx, domain.QuestionnaireAdminListFilter{Page: page, Limit: pageSize})
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

func (s *QuestionService) loadAllResponses(ctx context.Context, questionnaireID string) ([]domain.SurveyResponse, error) {
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

func validateQuestionSchema(schema domain.QuestionSchema) *apperror.AppError {
	if strings.TrimSpace(schema.Title) == "" {
		return apperror.BadRequest("schema.title不能为空")
	}
	switch schema.Type {
	case domain.QuestionTypeSingleChoice, domain.QuestionTypeMultipleChoice:
		if len(schema.Options) < 2 {
			return apperror.BadRequest("选择题至少需要2个选项")
		}
		seen := map[string]struct{}{}
		for _, opt := range schema.Options {
			if strings.TrimSpace(opt.OptionID) == "" || strings.TrimSpace(opt.Text) == "" {
				return apperror.BadRequest("选项ID与文本不能为空")
			}
			if _, ok := seen[opt.OptionID]; ok {
				return apperror.BadRequest("选项ID必须唯一")
			}
			seen[opt.OptionID] = struct{}{}
		}
	case domain.QuestionTypeText, domain.QuestionTypeNumber:
	default:
		return apperror.BadRequest("不支持的题型")
	}
	if schema.Validation.NumberType != "" && schema.Validation.NumberType != "integer" && schema.Validation.NumberType != "float" {
		return apperror.BadRequest("numberType仅支持integer或float")
	}
	if schema.Validation.MinVal != nil && schema.Validation.MaxVal != nil && *schema.Validation.MinVal > *schema.Validation.MaxVal {
		return apperror.BadRequest("minVal不能大于maxVal")
	}
	if schema.Validation.MinLength != nil && schema.Validation.MaxLength != nil && *schema.Validation.MinLength > *schema.Validation.MaxLength {
		return apperror.BadRequest("minLength不能大于maxLength")
	}
	if schema.Validation.MinSelect != nil && schema.Validation.MaxSelect != nil && *schema.Validation.MinSelect > *schema.Validation.MaxSelect {
		return apperror.BadRequest("minSelect不能大于maxSelect")
	}
	return nil
}

func normalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, tag := range tags {
		t := strings.TrimSpace(tag)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}
