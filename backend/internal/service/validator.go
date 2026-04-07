package service

import (
	"fmt"
	"math"
	"strings"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	"github.com/2018wzh/SimpleSurvey/backend/pkg/apperror"
)

func validateQuestionnaireInput(input CreateQuestionnaireInput) *apperror.AppError {
	details := map[string]string{}
	if strings.TrimSpace(input.Title) == "" {
		details["title"] = "问卷标题不能为空"
	}
	if len(input.Questions) == 0 {
		details["questions"] = "至少需要一个题目"
	}

	questionMap := map[string]domain.Question{}
	for i, q := range input.Questions {
		field := fmt.Sprintf("questions[%d]", i)
		if strings.TrimSpace(q.QuestionID) == "" {
			details[field+".questionId"] = "questionId不能为空"
			continue
		}
		if strings.TrimSpace(q.QuestionVersionID) == "" {
			details[field+".questionVersionId"] = "questionVersionId不能为空"
		}
		if q.Order < 0 {
			details[field+".order"] = "order不能为负数"
		}
		if _, exists := questionMap[q.QuestionID]; exists {
			details[field+".questionId"] = "questionId必须唯一"
			continue
		}
		if q.Snapshot == nil {
			details[field+".snapshot"] = "snapshot不能为空"
			questionMap[q.QuestionID] = q
			continue
		}

		schema := questionSchemaFromQuestion(q)
		if strings.TrimSpace(schema.Title) == "" {
			details[field+".title"] = "题目标题不能为空"
		}

		switch schema.Type {
		case domain.QuestionTypeSingleChoice, domain.QuestionTypeMultipleChoice:
			if len(schema.Options) < 2 {
				details[field+".options"] = "选择题至少需要2个选项"
			}
			optionIDs := map[string]struct{}{}
			for j, opt := range schema.Options {
				if strings.TrimSpace(opt.OptionID) == "" {
					details[fmt.Sprintf("%s.options[%d].optionId", field, j)] = "optionId不能为空"
				}
				if strings.TrimSpace(opt.Text) == "" {
					details[fmt.Sprintf("%s.options[%d].text", field, j)] = "选项文本不能为空"
				}
				if _, ok := optionIDs[opt.OptionID]; ok {
					details[field+".options"] = "选项ID必须唯一"
				}
				optionIDs[opt.OptionID] = struct{}{}
			}
			if schema.Type == domain.QuestionTypeMultipleChoice {
				if schema.Validation.MinSelect != nil && schema.Validation.MaxSelect != nil && *schema.Validation.MinSelect > *schema.Validation.MaxSelect {
					details[field+".validation"] = "minSelect不能大于maxSelect"
				}
			}
		case domain.QuestionTypeText:
			if schema.Validation.MinLength != nil && schema.Validation.MaxLength != nil && *schema.Validation.MinLength > *schema.Validation.MaxLength {
				details[field+".validation"] = "minLength不能大于maxLength"
			}
		case domain.QuestionTypeNumber:
			if schema.Validation.MinVal != nil && schema.Validation.MaxVal != nil && *schema.Validation.MinVal > *schema.Validation.MaxVal {
				details[field+".validation"] = "minVal不能大于maxVal"
			}
			if schema.Validation.NumberType != "" && schema.Validation.NumberType != "integer" && schema.Validation.NumberType != "float" {
				details[field+".validation.numberType"] = "numberType仅支持integer或float"
			}
		default:
			details[field+".type"] = "不支持的题型"
		}

		questionMap[q.QuestionID] = q
	}

	for i, rule := range input.LogicRules {
		field := fmt.Sprintf("logicRules[%d]", i)
		q, ok := questionMap[rule.ConditionQuestionID]
		if !ok {
			details[field+".conditionQuestionId"] = "引用了不存在的问题"
			continue
		}
		if q.Snapshot == nil {
			details[field+".conditionQuestionId"] = "引用题目缺少snapshot"
			continue
		}
		schema := questionSchemaFromQuestion(q)
		targetRaw, ok := rule.ActionDetails["targetQuestionId"]
		if !ok {
			details[field+".actionDetails"] = "必须包含targetQuestionId"
		} else {
			targetID, ok := targetRaw.(string)
			if !ok || strings.TrimSpace(targetID) == "" {
				details[field+".actionDetails.targetQuestionId"] = "targetQuestionId必须为非空字符串"
			} else if _, exists := questionMap[targetID]; !exists {
				details[field+".actionDetails.targetQuestionId"] = "targetQuestionId不存在"
			}
		}
		if rule.Action != domain.LogicActionJumpTo {
			details[field+".action"] = "当前仅支持JUMP_TO"
		}
		switch schema.Type {
		case domain.QuestionTypeSingleChoice:
			if rule.Operator != domain.LogicOperatorEquals {
				details[field+".operator"] = "单选题仅支持EQUALS"
			}
		case domain.QuestionTypeMultipleChoice:
			if rule.Operator != domain.LogicOperatorContains {
				details[field+".operator"] = "多选题仅支持CONTAINS"
			}
		case domain.QuestionTypeNumber:
			if rule.Operator != domain.LogicOperatorGreaterThan && rule.Operator != domain.LogicOperatorLessThan && rule.Operator != domain.LogicOperatorEquals {
				details[field+".operator"] = "数字题仅支持EQUALS/GREATER_THAN/LESS_THAN"
			}
		default:
			details[field+".operator"] = "该题型不支持跳转规则"
		}
	}

	if len(details) > 0 {
		return apperror.WithDetails(apperror.BadRequest("请求参数校验失败"), details)
	}
	return nil
}

func validateAnswers(questionnaire domain.Questionnaire, answers []domain.Answer) *apperror.AppError {
	if len(answers) == 0 {
		return apperror.BadRequest("answers不能为空")
	}

	questionMap := make(map[string]domain.Question, len(questionnaire.Questions))
	questionIDExists := make(map[string]struct{}, len(questionnaire.Questions))
	for _, q := range questionnaire.Questions {
		questionMap[questionRefKey(q.QuestionID, q.QuestionVersionID)] = q
		questionIDExists[q.QuestionID] = struct{}{}
	}

	answerMap := map[string]domain.Answer{}
	for _, answer := range answers {
		if strings.TrimSpace(answer.QuestionVersionID) == "" {
			return apperror.BadRequest(fmt.Sprintf("题目%s缺少questionVersionId", answer.QuestionID))
		}

		refKey := questionRefKey(answer.QuestionID, answer.QuestionVersionID)
		if _, exists := answerMap[refKey]; exists {
			return apperror.BadRequest(fmt.Sprintf("题目%s版本%s重复作答", answer.QuestionID, answer.QuestionVersionID))
		}
		q, ok := questionMap[refKey]
		if !ok {
			if _, exists := questionIDExists[answer.QuestionID]; exists {
				return apperror.PreconditionFailed(fmt.Sprintf("题目%s版本不匹配", answer.QuestionID))
			}
			return apperror.BadRequest(fmt.Sprintf("题目%s版本%s不存在", answer.QuestionID, answer.QuestionVersionID))
		}
		if err := validateSingleAnswer(q, answer.Value); err != nil {
			return err
		}
		answerMap[refKey] = answer
	}

	for _, q := range questionnaire.Questions {
		if questionIsRequired(q) {
			refKey := questionRefKey(q.QuestionID, q.QuestionVersionID)
			if _, ok := answerMap[refKey]; !ok {
				return apperror.BadRequest(fmt.Sprintf("题目%s版本%s为必答题", q.QuestionID, q.QuestionVersionID))
			}
		}
	}

	return nil
}

func validateSingleAnswer(q domain.Question, value interface{}) *apperror.AppError {
	if q.Snapshot == nil {
		return apperror.BadRequest(fmt.Sprintf("题目%s缺少snapshot", q.QuestionID))
	}
	schema := questionSchemaFromQuestion(q)
	switch schema.Type {
	case domain.QuestionTypeSingleChoice:
		answer, ok := value.(string)
		if !ok || strings.TrimSpace(answer) == "" {
			return apperror.BadRequest(fmt.Sprintf("题目%s必须选择一个选项", q.QuestionID))
		}
		allowed := map[string]struct{}{}
		for _, opt := range schema.Options {
			allowed[opt.OptionID] = struct{}{}
		}
		if _, ok := allowed[answer]; !ok {
			return apperror.BadRequest(fmt.Sprintf("题目%s选项非法", q.QuestionID))
		}
	case domain.QuestionTypeMultipleChoice:
		selections, ok := toStringSlice(value)
		if !ok {
			return apperror.BadRequest(fmt.Sprintf("题目%s必须为选项数组", q.QuestionID))
		}
		allowed := map[string]struct{}{}
		for _, opt := range schema.Options {
			allowed[opt.OptionID] = struct{}{}
		}
		for _, item := range selections {
			if _, exists := allowed[item]; !exists {
				return apperror.BadRequest(fmt.Sprintf("题目%s选项非法", q.QuestionID))
			}
		}
		if schema.Validation.MinSelect != nil && len(selections) < *schema.Validation.MinSelect {
			return apperror.BadRequest(fmt.Sprintf("题目%s至少选择%d项", q.QuestionID, *schema.Validation.MinSelect))
		}
		if schema.Validation.MaxSelect != nil && len(selections) > *schema.Validation.MaxSelect {
			return apperror.BadRequest(fmt.Sprintf("题目%s最多选择%d项", q.QuestionID, *schema.Validation.MaxSelect))
		}
	case domain.QuestionTypeText:
		text, ok := value.(string)
		if !ok {
			return apperror.BadRequest(fmt.Sprintf("题目%s必须为文本", q.QuestionID))
		}
		length := len([]rune(text))
		if schema.Validation.MinLength != nil && length < *schema.Validation.MinLength {
			return apperror.BadRequest(fmt.Sprintf("题目%s最少输入%d字", q.QuestionID, *schema.Validation.MinLength))
		}
		if schema.Validation.MaxLength != nil && length > *schema.Validation.MaxLength {
			return apperror.BadRequest(fmt.Sprintf("题目%s最多输入%d字", q.QuestionID, *schema.Validation.MaxLength))
		}
	case domain.QuestionTypeNumber:
		number, ok := toFloat64(value)
		if !ok {
			return apperror.BadRequest(fmt.Sprintf("题目%s必须为数字", q.QuestionID))
		}
		if schema.Validation.NumberType == "integer" && math.Mod(number, 1) != 0 {
			return apperror.BadRequest(fmt.Sprintf("题目%s必须为整数", q.QuestionID))
		}
		if schema.Validation.MinVal != nil && number < *schema.Validation.MinVal {
			return apperror.BadRequest(fmt.Sprintf("题目%s不能小于%v", q.QuestionID, *schema.Validation.MinVal))
		}
		if schema.Validation.MaxVal != nil && number > *schema.Validation.MaxVal {
			return apperror.BadRequest(fmt.Sprintf("题目%s不能大于%v", q.QuestionID, *schema.Validation.MaxVal))
		}
	default:
		return apperror.BadRequest(fmt.Sprintf("不支持的题型: %s", schema.Type))
	}
	return nil
}

func questionSchemaFromQuestion(q domain.Question) domain.QuestionSchema {
	if q.Snapshot == nil {
		return domain.QuestionSchema{}
	}
	return *q.Snapshot
}

func questionIsRequired(q domain.Question) bool {
	return q.Snapshot != nil && q.Snapshot.IsRequired
}

func toStringSlice(v interface{}) ([]string, bool) {
	switch value := v.(type) {
	case []string:
		return value, true
	case []interface{}:
		out := make([]string, 0, len(value))
		for _, item := range value {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	default:
		return nil, false
	}
}

func toFloat64(v interface{}) (float64, bool) {
	switch value := v.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int32:
		return float64(value), true
	case int64:
		return float64(value), true
	default:
		return 0, false
	}
}

func questionRefKey(questionID, questionVersionID string) string {
	return strings.TrimSpace(questionID) + "::" + strings.TrimSpace(questionVersionID)
}
