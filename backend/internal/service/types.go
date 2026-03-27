package service

import (
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
)

type CreateQuestionnaireInput struct {
	Title       string
	Description string
	Settings    domain.QuestionnaireSettings
	Questions   []domain.Question
	LogicRules  []domain.LogicRule
}

type UpdateQuestionnaireStatusInput struct {
	Status   domain.QuestionnaireStatus
	Deadline *time.Time
}

type SubmitResponseInput struct {
	IsAnonymous bool
	Answers     []domain.Answer
	Statistics  domain.ResponseStatistics
}

type QuestionnaireListItem struct {
	ID        string                     `json:"id"`
	Title     string                     `json:"title"`
	Status    domain.QuestionnaireStatus `json:"status"`
	CreatedAt time.Time                  `json:"createdAt"`
}
