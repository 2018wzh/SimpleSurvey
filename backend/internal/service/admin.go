package service

import (
	"context"
	"errors"
	"strings"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	"github.com/2018wzh/SimpleSurvey/backend/pkg/apperror"
)

type AdminService struct {
	users          domain.UserRepository
	questionnaires domain.QuestionnaireRepository
}

func NewAdminService(users domain.UserRepository, questionnaires domain.QuestionnaireRepository) *AdminService {
	return &AdminService{users: users, questionnaires: questionnaires}
}

func (s *AdminService) ListUsers(ctx context.Context, filter domain.UserListFilter) ([]domain.User, int64, *apperror.AppError) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	if filter.Role != "" && filter.Role != string(domain.UserRoleUser) && filter.Role != string(domain.UserRoleAdmin) {
		return nil, 0, apperror.BadRequest("角色参数不合法")
	}
	if filter.Status != "" && filter.Status != string(domain.UserStatusActive) && filter.Status != string(domain.UserStatusDisabled) {
		return nil, 0, apperror.BadRequest("状态参数不合法")
	}

	items, total, err := s.users.List(ctx, filter)
	if err != nil {
		return nil, 0, apperror.Internal("查询用户列表失败")
	}
	return items, total, nil
}

func (s *AdminService) UpdateUserRole(ctx context.Context, userID string, role domain.UserRole) *apperror.AppError {
	if role != domain.UserRoleUser && role != domain.UserRoleAdmin {
		return apperror.BadRequest("角色参数不合法")
	}

	err := s.users.UpdateRole(ctx, strings.TrimSpace(userID), role)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return apperror.NotFound("用户不存在")
		}
		return apperror.Internal("更新用户角色失败")
	}
	return nil
}

func (s *AdminService) UpdateUserStatus(ctx context.Context, userID string, status domain.UserStatus) *apperror.AppError {
	if status != domain.UserStatusActive && status != domain.UserStatusDisabled {
		return apperror.BadRequest("状态参数不合法")
	}

	err := s.users.UpdateStatus(ctx, strings.TrimSpace(userID), status)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return apperror.NotFound("用户不存在")
		}
		return apperror.Internal("更新用户状态失败")
	}
	return nil
}

func (s *AdminService) ListQuestionnaires(ctx context.Context, filter domain.QuestionnaireAdminListFilter) ([]domain.Questionnaire, int64, *apperror.AppError) {
	if filter.Status != "" {
		s := domain.QuestionnaireStatus(filter.Status)
		if s != domain.QuestionnaireStatusDraft && s != domain.QuestionnaireStatusPublished && s != domain.QuestionnaireStatusClosed {
			return nil, 0, apperror.BadRequest("问卷状态参数不合法")
		}
	}

	items, total, err := s.questionnaires.ListAll(ctx, filter)
	if err != nil {
		return nil, 0, apperror.Internal("查询问卷列表失败")
	}
	return items, total, nil
}

func (s *AdminService) UpdateQuestionnaireStatus(ctx context.Context, questionnaireID string, input UpdateQuestionnaireStatusInput) *apperror.AppError {
	if input.Status != domain.QuestionnaireStatusPublished && input.Status != domain.QuestionnaireStatusClosed && input.Status != domain.QuestionnaireStatusDraft {
		return apperror.BadRequest("问卷状态不合法")
	}

	err := s.questionnaires.UpdateStatusByAdmin(ctx, strings.TrimSpace(questionnaireID), input.Status, input.Deadline)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return apperror.NotFound("问卷不存在")
		}
		return apperror.Internal("更新问卷状态失败")
	}
	return nil
}
