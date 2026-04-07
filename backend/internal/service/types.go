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

type CreateQuestionInput struct {
	QuestionKey string
	Schema      domain.QuestionSchema
	Tags        []string
}

type CreateQuestionResult struct {
	ID        string `json:"id"`
	Version   int    `json:"version"`
	VersionID string `json:"versionId"`
}

type CreateQuestionVersionInput struct {
	BaseVersionID string
	ChangeType    domain.QuestionVersionChangeType
	Note          string
	Schema        domain.QuestionSchema
}

type RestoreQuestionVersionInput struct {
	FromVersionID string
	Note          string
}

type QuestionStatsInput struct {
	QuestionVersionID string
	From              *time.Time
	To                *time.Time
}

type CreateQuestionBankInput struct {
	Name        string
	Description string
	Visibility  domain.QuestionBankVisibility
	Items       []CreateQuestionBankItemInput
}

type CreateQuestionBankItemInput struct {
	QuestionID      string
	PinnedVersionID *string
	Order           int
}

type UpdateQuestionBankInput struct {
	Name        string
	Description string
	Visibility  domain.QuestionBankVisibility
}

type AddQuestionBankItemInput struct {
	QuestionID      string
	PinnedVersionID *string
	Order           int
}

type UpdateQuestionBankItemInput struct {
	PinnedVersionID *string
	Order           *int
}

type ShareQuestionBankInput struct {
	TargetUserID string
	Permission   domain.QuestionBankPermission
	ExpiresAt    *time.Time
}

type CrossTabDateRange struct {
	Start *time.Time
	End   *time.Time
}

type CrossTabFilters struct {
	DateRange        CrossTabDateRange
	CompletionStatus string
}

type CrossTabInput struct {
	RowQuestionID string
	ColQuestionID string
	Filters       CrossTabFilters
}
