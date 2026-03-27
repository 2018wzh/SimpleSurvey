package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	"github.com/2018wzh/SimpleSurvey/backend/pkg/auth"
)

type fakeIdentityUserRepo struct {
	created *domain.User
	byName  map[string]*domain.User
}

func (f *fakeIdentityUserRepo) Create(_ context.Context, user *domain.User) error {
	if user.ID == "" {
		user.ID = "507f1f77bcf86cd799439011"
	}
	copy := *user
	f.created = &copy
	if f.byName == nil {
		f.byName = map[string]*domain.User{}
	}
	f.byName[user.Username] = &copy
	return nil
}

func (f *fakeIdentityUserRepo) FindByUsername(_ context.Context, username string) (*domain.User, error) {
	if f.byName == nil {
		return nil, domain.ErrNotFound
	}
	user, ok := f.byName[username]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := *user
	return &copy, nil
}

func (f *fakeIdentityUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	for _, user := range f.byName {
		if user.ID == id {
			copy := *user
			return &copy, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeIdentityUserRepo) List(_ context.Context, _ domain.UserListFilter) ([]domain.User, int64, error) {
	items := make([]domain.User, 0, len(f.byName))
	for _, user := range f.byName {
		copy := *user
		items = append(items, copy)
	}
	return items, int64(len(items)), nil
}

func (f *fakeIdentityUserRepo) UpdateRole(_ context.Context, userID string, role domain.UserRole) error {
	for _, user := range f.byName {
		if user.ID == userID {
			user.Role = role
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeIdentityUserRepo) UpdateStatus(_ context.Context, userID string, status domain.UserStatus) error {
	for _, user := range f.byName {
		if user.ID == userID {
			user.Status = status
			return nil
		}
	}
	return domain.ErrNotFound
}

type fakeRefreshStore struct {
	items map[string]time.Time
}

func (f *fakeRefreshStore) Save(_ context.Context, userID, tokenID string, ttl time.Duration) error {
	if f.items == nil {
		f.items = map[string]time.Time{}
	}
	f.items[userID+":"+tokenID] = time.Now().Add(ttl)
	return nil
}

func (f *fakeRefreshStore) Exists(_ context.Context, userID, tokenID string) (bool, error) {
	expiresAt, ok := f.items[userID+":"+tokenID]
	if !ok {
		return false, nil
	}
	if time.Now().After(expiresAt) {
		delete(f.items, userID+":"+tokenID)
		return false, nil
	}
	return true, nil
}

func (f *fakeRefreshStore) Delete(_ context.Context, userID, tokenID string) error {
	delete(f.items, userID+":"+tokenID)
	return nil
}

func TestRegisterStoresArgon2Hash(t *testing.T) {
	repo := &fakeIdentityUserRepo{}
	store := &fakeRefreshStore{}
	svc := NewIdentityService(repo, store, "secret", time.Hour, 24*time.Hour)

	_, appErr := svc.Register(context.Background(), "alice", "strong-password")
	if appErr != nil {
		t.Fatalf("expected register success, got err: %v", appErr)
	}
	if repo.created == nil {
		t.Fatal("expected user to be created")
	}
	if !strings.HasPrefix(repo.created.Password, "$argon2id$") {
		t.Fatalf("expected argon2 hash, got: %s", repo.created.Password)
	}
}

func TestLoginAndRefreshFlow(t *testing.T) {
	hashed, err := auth.HashPassword("strong-password")
	if err != nil {
		t.Fatalf("hash password failed: %v", err)
	}

	repo := &fakeIdentityUserRepo{
		byName: map[string]*domain.User{
			"alice": {
				ID:       "507f1f77bcf86cd799439011",
				Username: "alice",
				Password: hashed,
				Role:     domain.UserRoleUser,
				Status:   domain.UserStatusActive,
			},
		},
	}
	store := &fakeRefreshStore{}
	svc := NewIdentityService(repo, store, "secret", time.Hour, 24*time.Hour)

	tokens, appErr := svc.Login(context.Background(), "alice", "strong-password")
	if appErr != nil {
		t.Fatalf("expected login success, got err: %v", appErr)
	}
	if tokens.Token == "" || tokens.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}

	accessClaims, err := auth.ParseToken("secret", tokens.Token)
	if err != nil {
		t.Fatalf("parse access token failed: %v", err)
	}
	if accessClaims.TokenType != auth.TokenTypeAccess {
		t.Fatalf("expected access token type, got: %s", accessClaims.TokenType)
	}

	refreshClaims, err := auth.ParseToken("secret", tokens.RefreshToken)
	if err != nil {
		t.Fatalf("parse refresh token failed: %v", err)
	}
	if refreshClaims.TokenType != auth.TokenTypeRefresh {
		t.Fatalf("expected refresh token type, got: %s", refreshClaims.TokenType)
	}
	if refreshClaims.ID == "" {
		t.Fatal("expected refresh token to contain token id")
	}

	exists, _ := store.Exists(context.Background(), refreshClaims.UserID, refreshClaims.ID)
	if !exists {
		t.Fatal("expected refresh token to be stored")
	}

	newTokens, appErr := svc.Refresh(context.Background(), tokens.RefreshToken)
	if appErr != nil {
		t.Fatalf("expected refresh success, got err: %v", appErr)
	}
	if newTokens.RefreshToken == tokens.RefreshToken {
		t.Fatal("expected refresh token rotation")
	}

	_, appErr = svc.Refresh(context.Background(), tokens.RefreshToken)
	if appErr == nil {
		t.Fatal("expected old refresh token to be invalid after rotation")
	}
}
