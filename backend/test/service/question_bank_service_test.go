package service

import (
	"context"
	"testing"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	. "github.com/2018wzh/SimpleSurvey/backend/internal/service"
)

type fakeQuestionBankRepo struct {
	items map[string]domain.QuestionBank
}

func newFakeQuestionBankRepo() *fakeQuestionBankRepo {
	return &fakeQuestionBankRepo{items: map[string]domain.QuestionBank{}}
}

func (f *fakeQuestionBankRepo) Create(_ context.Context, bank *domain.QuestionBank) error {
	if bank.ID == "" {
		bank.ID = "67f3e5f244f95a7d05b5a701"
	}
	f.items[bank.ID] = *bank
	return nil
}

func (f *fakeQuestionBankRepo) FindByID(_ context.Context, id string) (*domain.QuestionBank, error) {
	item, ok := f.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := item
	return &copy, nil
}

func (f *fakeQuestionBankRepo) FindByIDForUser(_ context.Context, id, userID string) (*domain.QuestionBank, error) {
	item, ok := f.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if item.OwnerID == userID {
		copy := item
		return &copy, nil
	}
	for _, share := range item.SharedWith {
		if share.UserID == userID {
			copy := item
			return &copy, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeQuestionBankRepo) ListByOwnerOrShared(_ context.Context, userID string, _ domain.QuestionBankListFilter) ([]domain.QuestionBank, int64, error) {
	items := make([]domain.QuestionBank, 0)
	for _, item := range f.items {
		if item.OwnerID == userID {
			items = append(items, item)
			continue
		}
		for _, share := range item.SharedWith {
			if share.UserID == userID {
				items = append(items, item)
				break
			}
		}
	}
	return items, int64(len(items)), nil
}

func (f *fakeQuestionBankRepo) UpdateBase(_ context.Context, bank *domain.QuestionBank) error {
	if _, ok := f.items[bank.ID]; !ok {
		return domain.ErrNotFound
	}
	f.items[bank.ID] = *bank
	return nil
}

func (f *fakeQuestionBankRepo) UpdateItems(_ context.Context, bankID string, items []domain.QuestionBankItem) error {
	bank, ok := f.items[bankID]
	if !ok {
		return domain.ErrNotFound
	}
	bank.Items = items
	bank.UpdatedAt = time.Now().UTC()
	f.items[bankID] = bank
	return nil
}

func (f *fakeQuestionBankRepo) UpdateShares(_ context.Context, bankID string, shares []domain.QuestionBankShare) error {
	bank, ok := f.items[bankID]
	if !ok {
		return domain.ErrNotFound
	}
	bank.SharedWith = shares
	bank.UpdatedAt = time.Now().UTC()
	f.items[bankID] = bank
	return nil
}

func TestQuestionBankServiceFlow(t *testing.T) {
	repo := newFakeQuestionBankRepo()
	svc := NewQuestionBankService(repo)

	bankID, appErr := svc.Create(context.Background(), "507f1f77bcf86cd799439011", CreateQuestionBankInput{
		Name:       "基础题库",
		Visibility: domain.QuestionBankVisibilityTeam,
		Items:      []CreateQuestionBankItemInput{{QuestionID: "67f3e5f244f95a7d05b5a111", Order: 1}},
	})
	if appErr != nil {
		t.Fatalf("expected create bank success, got appErr=%v", appErr)
	}

	items, appErr := svc.AddItem(context.Background(), "507f1f77bcf86cd799439011", bankID, AddQuestionBankItemInput{QuestionID: "67f3e5f244f95a7d05b5a112", Order: 2})
	if appErr != nil {
		t.Fatalf("expected add item success, got appErr=%v", appErr)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	shares, appErr := svc.Share(context.Background(), "507f1f77bcf86cd799439011", bankID, ShareQuestionBankInput{TargetUserID: "507f1f77bcf86cd799439099", Permission: domain.QuestionBankPermissionManage})
	if appErr != nil {
		t.Fatalf("expected share success, got appErr=%v", appErr)
	}
	if len(shares) != 1 {
		t.Fatalf("expected 1 share, got %d", len(shares))
	}

	_, appErr = svc.Unshare(context.Background(), "507f1f77bcf86cd799439011", bankID, "507f1f77bcf86cd799439099")
	if appErr != nil {
		t.Fatalf("expected unshare success, got appErr=%v", appErr)
	}
}

func TestQuestionBankServiceSharePermissionBoundaries(t *testing.T) {
	repo := newFakeQuestionBankRepo()
	svc := NewQuestionBankService(repo)
	ownerID := "507f1f77bcf86cd799439011"
	managerID := "507f1f77bcf86cd799439012"
	useOnlyID := "507f1f77bcf86cd799439013"

	bankID, appErr := svc.Create(context.Background(), ownerID, CreateQuestionBankInput{
		Name:  "团队题库",
		Items: []CreateQuestionBankItemInput{{QuestionID: "67f3e5f244f95a7d05b5a111", Order: 1}},
	})
	if appErr != nil {
		t.Fatalf("expected create bank success, got appErr=%v", appErr)
	}

	_, appErr = svc.Share(context.Background(), ownerID, bankID, ShareQuestionBankInput{TargetUserID: useOnlyID, Permission: domain.QuestionBankPermissionUse})
	if appErr != nil {
		t.Fatalf("expected share use permission success, got appErr=%v", appErr)
	}

	_, appErr = svc.AddItem(context.Background(), useOnlyID, bankID, AddQuestionBankItemInput{QuestionID: "67f3e5f244f95a7d05b5a112", Order: 2})
	if appErr == nil || appErr.Code != 403 {
		t.Fatalf("expected forbidden for use-only member managing bank, got %+v", appErr)
	}

	expired := time.Now().UTC().Add(-time.Minute)
	_, appErr = svc.Share(context.Background(), ownerID, bankID, ShareQuestionBankInput{TargetUserID: managerID, Permission: domain.QuestionBankPermissionManage, ExpiresAt: &expired})
	if appErr != nil {
		t.Fatalf("expected share manage (expired) success, got appErr=%v", appErr)
	}

	_, appErr = svc.UpdateBase(context.Background(), managerID, bankID, UpdateQuestionBankInput{Name: "过期管理员修改", Visibility: domain.QuestionBankVisibilityPrivate})
	if appErr == nil || appErr.Code != 403 {
		t.Fatalf("expected forbidden for expired manager permission, got %+v", appErr)
	}

	validUntil := time.Now().UTC().Add(time.Hour)
	_, appErr = svc.Share(context.Background(), ownerID, bankID, ShareQuestionBankInput{TargetUserID: managerID, Permission: domain.QuestionBankPermissionManage, ExpiresAt: &validUntil})
	if appErr != nil {
		t.Fatalf("expected share manage success, got appErr=%v", appErr)
	}

	items, appErr := svc.AddItem(context.Background(), managerID, bankID, AddQuestionBankItemInput{QuestionID: "67f3e5f244f95a7d05b5a113", Order: 3})
	if appErr != nil {
		t.Fatalf("expected manager add item success, got appErr=%v", appErr)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items after manager add, got %d", len(items))
	}
}

func TestQuestionBankServiceShareUpsertAndList(t *testing.T) {
	repo := newFakeQuestionBankRepo()
	svc := NewQuestionBankService(repo)
	ownerID := "507f1f77bcf86cd799439011"
	targetUserID := "507f1f77bcf86cd799439099"

	bankID, appErr := svc.Create(context.Background(), ownerID, CreateQuestionBankInput{Name: "共享题库"})
	if appErr != nil {
		t.Fatalf("expected create bank success, got appErr=%v", appErr)
	}

	shares, appErr := svc.Share(context.Background(), ownerID, bankID, ShareQuestionBankInput{TargetUserID: targetUserID, Permission: domain.QuestionBankPermissionUse})
	if appErr != nil {
		t.Fatalf("expected first share success, got appErr=%v", appErr)
	}
	if len(shares) != 1 || shares[0].Permission != domain.QuestionBankPermissionUse {
		t.Fatalf("expected one use share, got %+v", shares)
	}

	shares, appErr = svc.Share(context.Background(), ownerID, bankID, ShareQuestionBankInput{TargetUserID: targetUserID, Permission: domain.QuestionBankPermissionManage})
	if appErr != nil {
		t.Fatalf("expected second share(upsert) success, got appErr=%v", appErr)
	}
	if len(shares) != 1 || shares[0].Permission != domain.QuestionBankPermissionManage {
		t.Fatalf("expected upserted manage share, got %+v", shares)
	}

	banks, total, appErr := svc.List(context.Background(), targetUserID, domain.QuestionBankListFilter{Page: 1, Limit: 20})
	if appErr != nil {
		t.Fatalf("expected shared user list success, got appErr=%v", appErr)
	}
	if total != 1 || len(banks) != 1 || banks[0].ID != bankID {
		t.Fatalf("expected shared user sees exactly one bank %s, got total=%d items=%d", bankID, total, len(banks))
	}
}
