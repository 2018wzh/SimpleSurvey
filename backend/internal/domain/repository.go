package domain

import (
	"context"
	"time"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	List(ctx context.Context, filter UserListFilter) ([]User, int64, error)
	UpdateRole(ctx context.Context, userID string, role UserRole) error
	UpdateStatus(ctx context.Context, userID string, status UserStatus) error
	UpdatePassword(ctx context.Context, userID string, password string) error
}

type QuestionnaireRepository interface {
	Create(ctx context.Context, questionnaire *Questionnaire) error
	FindByID(ctx context.Context, id string) (*Questionnaire, error)
	FindByIDAndCreator(ctx context.Context, id, creatorID string) (*Questionnaire, error)
	ListByCreator(ctx context.Context, creatorID string, filter QuestionnaireListFilter) ([]Questionnaire, int64, error)
	ListAll(ctx context.Context, filter QuestionnaireAdminListFilter) ([]Questionnaire, int64, error)
	UpdateStatus(ctx context.Context, id, creatorID string, status QuestionnaireStatus, deadline *time.Time) error
	UpdateStatusByAdmin(ctx context.Context, id string, status QuestionnaireStatus, deadline *time.Time) error
}

type ResponseRepository interface {
	Create(ctx context.Context, response *SurveyResponse) error
	ListByQuestionnaire(ctx context.Context, questionnaireID string, filter ResponseListFilter) ([]SurveyResponse, int64, error)
	CountByQuestionnaire(ctx context.Context, questionnaireID string) (int64, error)
}

type QuestionRepository interface {
	Create(ctx context.Context, question *QuestionEntity, version *QuestionVersion) error
	FindByID(ctx context.Context, id string) (*QuestionEntity, error)
	FindByIDAndOwner(ctx context.Context, id, ownerID string) (*QuestionEntity, error)
	FindByQuestionKey(ctx context.Context, questionKey string) (*QuestionEntity, error)
	FindVersionByID(ctx context.Context, versionID string) (*QuestionVersion, error)
	ListVersions(ctx context.Context, questionID string) ([]QuestionVersion, error)
	CreateVersion(ctx context.Context, question *QuestionEntity, version *QuestionVersion) error
	ListByOwner(ctx context.Context, ownerID string, filter QuestionListFilter) ([]QuestionEntity, int64, error)
}

type QuestionBankRepository interface {
	Create(ctx context.Context, bank *QuestionBank) error
	FindByID(ctx context.Context, id string) (*QuestionBank, error)
	FindByIDForUser(ctx context.Context, id, userID string) (*QuestionBank, error)
	ListByOwnerOrShared(ctx context.Context, userID string, filter QuestionBankListFilter) ([]QuestionBank, int64, error)
	UpdateBase(ctx context.Context, bank *QuestionBank) error
	UpdateItems(ctx context.Context, bankID string, items []QuestionBankItem) error
	UpdateShares(ctx context.Context, bankID string, shares []QuestionBankShare) error
}
