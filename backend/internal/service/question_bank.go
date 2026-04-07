package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	"github.com/2018wzh/SimpleSurvey/backend/pkg/apperror"
)

type QuestionBankService struct {
	banks domain.QuestionBankRepository
}

func NewQuestionBankService(banks domain.QuestionBankRepository) *QuestionBankService {
	return &QuestionBankService{banks: banks}
}

func (s *QuestionBankService) Create(ctx context.Context, ownerID string, input CreateQuestionBankInput) (string, *apperror.AppError) {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return "", apperror.Unauthorized("未授权")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return "", apperror.BadRequest("题库名称不能为空")
	}
	visibility := input.Visibility
	if visibility == "" {
		visibility = domain.QuestionBankVisibilityPrivate
	}
	if visibility != domain.QuestionBankVisibilityPrivate && visibility != domain.QuestionBankVisibilityTeam {
		return "", apperror.BadRequest("visibility仅支持private/team")
	}

	now := time.Now().UTC()
	items := make([]domain.QuestionBankItem, 0, len(input.Items))
	for _, item := range input.Items {
		questionID := strings.TrimSpace(item.QuestionID)
		if questionID == "" {
			return "", apperror.BadRequest("items.questionId不能为空")
		}
		order := item.Order
		if order <= 0 {
			order = len(items) + 1
		}
		items = append(items, domain.QuestionBankItem{
			QuestionID:      questionID,
			PinnedVersionID: item.PinnedVersionID,
			AddedBy:         ownerID,
			AddedAt:         now,
			Order:           order,
		})
	}

	bank := &domain.QuestionBank{
		Name:        name,
		OwnerID:     ownerID,
		Description: strings.TrimSpace(input.Description),
		Visibility:  visibility,
		Items:       items,
		SharedWith:  []domain.QuestionBankShare{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.banks.Create(ctx, bank); err != nil {
		return "", apperror.Internal("创建题库失败")
	}
	return bank.ID, nil
}

func (s *QuestionBankService) List(ctx context.Context, userID string, filter domain.QuestionBankListFilter) ([]domain.QuestionBank, int64, *apperror.AppError) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, 0, apperror.Unauthorized("未授权")
	}
	items, total, err := s.banks.ListByOwnerOrShared(ctx, userID, filter)
	if err != nil {
		return nil, 0, apperror.Internal("查询题库失败")
	}
	return items, total, nil
}

func (s *QuestionBankService) UpdateBase(ctx context.Context, userID, bankID string, input UpdateQuestionBankInput) (*domain.QuestionBank, *apperror.AppError) {
	bank, appErr := s.getAuthorizedManageBank(ctx, userID, bankID)
	if appErr != nil {
		return nil, appErr
	}
	bank.Name = strings.TrimSpace(input.Name)
	if bank.Name == "" {
		return nil, apperror.BadRequest("题库名称不能为空")
	}
	bank.Description = strings.TrimSpace(input.Description)
	if input.Visibility != "" {
		if input.Visibility != domain.QuestionBankVisibilityPrivate && input.Visibility != domain.QuestionBankVisibilityTeam {
			return nil, apperror.BadRequest("visibility仅支持private/team")
		}
		bank.Visibility = input.Visibility
	}
	if err := s.banks.UpdateBase(ctx, bank); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("题库不存在")
		}
		return nil, apperror.Internal("更新题库失败")
	}
	updated, err := s.banks.FindByIDForUser(ctx, bankID, strings.TrimSpace(userID))
	if err != nil {
		return nil, apperror.Internal("查询题库失败")
	}
	return updated, nil
}

func (s *QuestionBankService) AddItem(ctx context.Context, userID, bankID string, input AddQuestionBankItemInput) ([]domain.QuestionBankItem, *apperror.AppError) {
	bank, appErr := s.getAuthorizedManageBank(ctx, userID, bankID)
	if appErr != nil {
		return nil, appErr
	}
	questionID := strings.TrimSpace(input.QuestionID)
	if questionID == "" {
		return nil, apperror.BadRequest("questionId不能为空")
	}
	for _, item := range bank.Items {
		if item.QuestionID == questionID {
			return nil, apperror.Conflict("题目已在题库中")
		}
	}
	order := input.Order
	if order <= 0 {
		order = len(bank.Items) + 1
	}
	bank.Items = append(bank.Items, domain.QuestionBankItem{
		QuestionID:      questionID,
		PinnedVersionID: input.PinnedVersionID,
		AddedBy:         strings.TrimSpace(userID),
		AddedAt:         time.Now().UTC(),
		Order:           order,
	})
	if err := s.banks.UpdateItems(ctx, bank.ID, bank.Items); err != nil {
		return nil, apperror.Internal("更新题库题目失败")
	}
	return bank.Items, nil
}

func (s *QuestionBankService) UpdateItem(ctx context.Context, userID, bankID, questionID string, input UpdateQuestionBankItemInput) ([]domain.QuestionBankItem, *apperror.AppError) {
	bank, appErr := s.getAuthorizedManageBank(ctx, userID, bankID)
	if appErr != nil {
		return nil, appErr
	}
	questionID = strings.TrimSpace(questionID)
	updated := false
	for i := range bank.Items {
		if bank.Items[i].QuestionID != questionID {
			continue
		}
		if input.PinnedVersionID != nil {
			bank.Items[i].PinnedVersionID = input.PinnedVersionID
		}
		if input.Order != nil && *input.Order > 0 {
			bank.Items[i].Order = *input.Order
		}
		updated = true
		break
	}
	if !updated {
		return nil, apperror.NotFound("题库中不存在该题目")
	}
	if err := s.banks.UpdateItems(ctx, bank.ID, bank.Items); err != nil {
		return nil, apperror.Internal("更新题库题目失败")
	}
	return bank.Items, nil
}

func (s *QuestionBankService) RemoveItem(ctx context.Context, userID, bankID, questionID string) ([]domain.QuestionBankItem, *apperror.AppError) {
	bank, appErr := s.getAuthorizedManageBank(ctx, userID, bankID)
	if appErr != nil {
		return nil, appErr
	}
	questionID = strings.TrimSpace(questionID)
	items := make([]domain.QuestionBankItem, 0, len(bank.Items))
	removed := false
	for _, item := range bank.Items {
		if item.QuestionID == questionID {
			removed = true
			continue
		}
		items = append(items, item)
	}
	if !removed {
		return nil, apperror.NotFound("题库中不存在该题目")
	}
	bank.Items = items
	if err := s.banks.UpdateItems(ctx, bank.ID, bank.Items); err != nil {
		return nil, apperror.Internal("更新题库题目失败")
	}
	return bank.Items, nil
}

func (s *QuestionBankService) Share(ctx context.Context, userID, bankID string, input ShareQuestionBankInput) ([]domain.QuestionBankShare, *apperror.AppError) {
	bank, appErr := s.getAuthorizedManageBank(ctx, userID, bankID)
	if appErr != nil {
		return nil, appErr
	}
	targetUserID := strings.TrimSpace(input.TargetUserID)
	if targetUserID == "" {
		return nil, apperror.BadRequest("targetUserId不能为空")
	}
	permission := input.Permission
	if permission == "" {
		permission = domain.QuestionBankPermissionUse
	}
	if permission != domain.QuestionBankPermissionUse && permission != domain.QuestionBankPermissionManage {
		return nil, apperror.BadRequest("permission仅支持use/manage")
	}

	now := time.Now().UTC()
	exists := false
	for i := range bank.SharedWith {
		if bank.SharedWith[i].UserID != targetUserID {
			continue
		}
		bank.SharedWith[i].Permission = permission
		bank.SharedWith[i].GrantedBy = strings.TrimSpace(userID)
		bank.SharedWith[i].GrantedAt = now
		bank.SharedWith[i].ExpiresAt = input.ExpiresAt
		exists = true
		break
	}
	if !exists {
		bank.SharedWith = append(bank.SharedWith, domain.QuestionBankShare{
			UserID:     targetUserID,
			Permission: permission,
			GrantedBy:  strings.TrimSpace(userID),
			GrantedAt:  now,
			ExpiresAt:  input.ExpiresAt,
		})
	}
	if err := s.banks.UpdateShares(ctx, bank.ID, bank.SharedWith); err != nil {
		return nil, apperror.Internal("更新题库共享失败")
	}
	return bank.SharedWith, nil
}

func (s *QuestionBankService) Unshare(ctx context.Context, userID, bankID, targetUserID string) ([]domain.QuestionBankShare, *apperror.AppError) {
	bank, appErr := s.getAuthorizedManageBank(ctx, userID, bankID)
	if appErr != nil {
		return nil, appErr
	}
	targetUserID = strings.TrimSpace(targetUserID)
	shares := make([]domain.QuestionBankShare, 0, len(bank.SharedWith))
	removed := false
	for _, share := range bank.SharedWith {
		if share.UserID == targetUserID {
			removed = true
			continue
		}
		shares = append(shares, share)
	}
	if !removed {
		return nil, apperror.NotFound("未找到共享关系")
	}
	bank.SharedWith = shares
	if err := s.banks.UpdateShares(ctx, bank.ID, bank.SharedWith); err != nil {
		return nil, apperror.Internal("更新题库共享失败")
	}
	return bank.SharedWith, nil
}

func (s *QuestionBankService) getAuthorizedManageBank(ctx context.Context, userID, bankID string) (*domain.QuestionBank, *apperror.AppError) {
	userID = strings.TrimSpace(userID)
	bankID = strings.TrimSpace(bankID)
	if userID == "" {
		return nil, apperror.Unauthorized("未授权")
	}
	bank, err := s.banks.FindByIDForUser(ctx, bankID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("题库不存在")
		}
		return nil, apperror.Internal("查询题库失败")
	}
	if bank.OwnerID == userID {
		return bank, nil
	}
	for _, share := range bank.SharedWith {
		if share.UserID == userID && share.Permission == domain.QuestionBankPermissionManage {
			if share.ExpiresAt != nil && share.ExpiresAt.Before(time.Now().UTC()) {
				continue
			}
			return bank, nil
		}
	}
	return nil, apperror.Forbidden("无管理权限")
}
