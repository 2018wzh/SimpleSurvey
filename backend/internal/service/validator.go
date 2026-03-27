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
		if _, exists := questionMap[q.QuestionID]; exists {
			details[field+".questionId"] = "questionId必须唯一"
			continue
		}
		if strings.TrimSpace(q.Title) == "" {
			details[field+".title"] = "题目标题不能为空"
		}

		switch q.Type {
		case domain.QuestionTypeSingleChoice, domain.QuestionTypeMultipleChoice:
			if len(q.Options) < 2 {
				details[field+".options"] = "选择题至少需要2个选项"
			}
			optionIDs := map[string]struct{}{}
			for j, opt := range q.Options {
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
			if q.Type == domain.QuestionTypeMultipleChoice {
				if q.Validation.MinSelect != nil && q.Validation.MaxSelect != nil && *q.Validation.MinSelect > *q.Validation.MaxSelect {
					details[field+".validation"] = "minSelect不能大于maxSelect"
				}
			}
		case domain.QuestionTypeText:
			if q.Validation.MinLength != nil && q.Validation.MaxLength != nil && *q.Validation.MinLength > *q.Validation.MaxLength {
				details[field+".validation"] = "minLength不能大于maxLength"
			}
		case domain.QuestionTypeNumber:
			if q.Validation.MinVal != nil && q.Validation.MaxVal != nil && *q.Validation.MinVal > *q.Validation.MaxVal {
				details[field+".validation"] = "minVal不能大于maxVal"
			}
			if q.Validation.NumberType != "" && q.Validation.NumberType != "integer" && q.Validation.NumberType != "float" {
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
		switch q.Type {
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
	for _, q := range questionnaire.Questions {
		questionMap[q.QuestionID] = q
	}

	answerMap := map[string]domain.Answer{}
	for _, answer := range answers {
		if _, exists := answerMap[answer.QuestionID]; exists {
			return apperror.BadRequest(fmt.Sprintf("题目%s重复作答", answer.QuestionID))
		}
		q, ok := questionMap[answer.QuestionID]
		if !ok {
			return apperror.BadRequest(fmt.Sprintf("题目%s不存在", answer.QuestionID))
		}
		if err := validateSingleAnswer(q, answer.Value); err != nil {
			return err
		}
		answerMap[answer.QuestionID] = answer
	}

	for _, q := range questionnaire.Questions {
		if q.IsRequired {
			if _, ok := answerMap[q.QuestionID]; !ok {
				return apperror.BadRequest(fmt.Sprintf("题目%s为必答题", q.QuestionID))
			}
		}
	}

	return nil
}

func validateSingleAnswer(q domain.Question, value interface{}) *apperror.AppError {
	switch q.Type {
	case domain.QuestionTypeSingleChoice:
		answer, ok := value.(string)
		if !ok || strings.TrimSpace(answer) == "" {
			return apperror.BadRequest(fmt.Sprintf("题目%s必须选择一个选项", q.QuestionID))
		}
		allowed := map[string]struct{}{}
		for _, opt := range q.Options {
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
		for _, opt := range q.Options {
			allowed[opt.OptionID] = struct{}{}
		}
		for _, item := range selections {
			if _, exists := allowed[item]; !exists {
				return apperror.BadRequest(fmt.Sprintf("题目%s选项非法", q.QuestionID))
			}
		}
		if q.Validation.MinSelect != nil && len(selections) < *q.Validation.MinSelect {
			return apperror.BadRequest(fmt.Sprintf("题目%s至少选择%d项", q.QuestionID, *q.Validation.MinSelect))
		}
		if q.Validation.MaxSelect != nil && len(selections) > *q.Validation.MaxSelect {
			return apperror.BadRequest(fmt.Sprintf("题目%s最多选择%d项", q.QuestionID, *q.Validation.MaxSelect))
		}
	case domain.QuestionTypeText:
		text, ok := value.(string)
		if !ok {
			return apperror.BadRequest(fmt.Sprintf("题目%s必须为文本", q.QuestionID))
		}
		length := len([]rune(text))
		if q.Validation.MinLength != nil && length < *q.Validation.MinLength {
			return apperror.BadRequest(fmt.Sprintf("题目%s最少输入%d字", q.QuestionID, *q.Validation.MinLength))
		}
		if q.Validation.MaxLength != nil && length > *q.Validation.MaxLength {
			return apperror.BadRequest(fmt.Sprintf("题目%s最多输入%d字", q.QuestionID, *q.Validation.MaxLength))
		}
	case domain.QuestionTypeNumber:
		number, ok := toFloat64(value)
		if !ok {
			return apperror.BadRequest(fmt.Sprintf("题目%s必须为数字", q.QuestionID))
		}
		if q.Validation.NumberType == "integer" && math.Mod(number, 1) != 0 {
			return apperror.BadRequest(fmt.Sprintf("题目%s必须为整数", q.QuestionID))
		}
		if q.Validation.MinVal != nil && number < *q.Validation.MinVal {
			return apperror.BadRequest(fmt.Sprintf("题目%s不能小于%v", q.QuestionID, *q.Validation.MinVal))
		}
		if q.Validation.MaxVal != nil && number > *q.Validation.MaxVal {
			return apperror.BadRequest(fmt.Sprintf("题目%s不能大于%v", q.QuestionID, *q.Validation.MaxVal))
		}
	default:
		return apperror.BadRequest(fmt.Sprintf("不支持的题型: %s", q.Type))
	}
	return nil
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
