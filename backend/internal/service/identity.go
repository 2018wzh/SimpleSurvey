package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	"github.com/2018wzh/SimpleSurvey/backend/pkg/apperror"
	"github.com/2018wzh/SimpleSurvey/backend/pkg/auth"
	"github.com/google/uuid"
)

type RefreshTokenStore interface {
	Save(ctx context.Context, userID, tokenID string, ttl time.Duration) error
	Exists(ctx context.Context, userID, tokenID string) (bool, error)
	Delete(ctx context.Context, userID, tokenID string) error
}

type AuthTokens struct {
	Token            string `json:"token"`
	ExpiresIn        int    `json:"expiresIn"`
	RefreshToken     string `json:"refreshToken"`
	RefreshExpiresIn int    `json:"refreshExpiresIn"`
}

type IdentityService struct {
	users          domain.UserRepository
	refreshStore   RefreshTokenStore
	jwtSecret      string
	accessExpires  time.Duration
	refreshExpires time.Duration
}

func NewIdentityService(users domain.UserRepository, refreshStore RefreshTokenStore, jwtSecret string, accessExpires, refreshExpires time.Duration) *IdentityService {
	return &IdentityService{
		users:          users,
		refreshStore:   refreshStore,
		jwtSecret:      jwtSecret,
		accessExpires:  accessExpires,
		refreshExpires: refreshExpires,
	}
}

func (s *IdentityService) Register(ctx context.Context, username, password string) (string, *apperror.AppError) {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 50 {
		return "", apperror.BadRequest("用户名长度必须在3~50之间")
	}
	if len(password) < 8 || len(password) > 128 {
		return "", apperror.BadRequest("密码长度必须在8~128之间")
	}

	hashed, err := auth.HashPassword(password)
	if err != nil {
		return "", apperror.Internal("密码处理失败")
	}

	user := &domain.User{
		Username:  username,
		Password:  hashed,
		CreatedAt: time.Now().UTC(),
		Role:      domain.UserRoleUser,
		Status:    domain.UserStatusActive,
	}

	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, domain.ErrDuplicate) {
			return "", apperror.Conflict("用户名已存在")
		}
		return "", apperror.Internal("注册失败")
	}
	return user.ID, nil
}

func (s *IdentityService) Login(ctx context.Context, username, password string) (*AuthTokens, *apperror.AppError) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, apperror.BadRequest("用户名和密码不能为空")
	}

	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.Unauthorized("用户名或密码错误")
		}
		return nil, apperror.Internal("登录失败")
	}

	ok, err := auth.VerifyPassword(user.Password, password)
	if err != nil || !ok {
		return nil, apperror.Unauthorized("用户名或密码错误")
	}
	if user.Status != domain.UserStatusActive {
		return nil, apperror.Forbidden("账号已被禁用")
	}

	tokens, err := s.issueTokenPair(ctx, user.ID, user.Username, string(user.Role))
	if err != nil {
		return nil, apperror.Internal("生成Token失败")
	}
	return tokens, nil
}

func (s *IdentityService) Refresh(ctx context.Context, refreshToken string) (*AuthTokens, *apperror.AppError) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, apperror.BadRequest("refreshToken不能为空")
	}

	claims, err := auth.ParseToken(s.jwtSecret, refreshToken)
	if err != nil {
		return nil, apperror.Unauthorized("refreshToken无效或已过期")
	}
	if claims.TokenType != auth.TokenTypeRefresh {
		return nil, apperror.Unauthorized("token类型错误")
	}
	if claims.ID == "" {
		return nil, apperror.Unauthorized("refreshToken缺少标识")
	}

	exists, err := s.refreshStore.Exists(ctx, claims.UserID, claims.ID)
	if err != nil {
		return nil, apperror.Internal("refreshToken校验失败")
	}
	if !exists {
		return nil, apperror.Unauthorized("refreshToken已失效")
	}

	_ = s.refreshStore.Delete(ctx, claims.UserID, claims.ID)

	user, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.Unauthorized("用户不存在")
		}
		return nil, apperror.Internal("刷新Token失败")
	}
	if user.Status != domain.UserStatusActive {
		return nil, apperror.Forbidden("账号已被禁用")
	}

	tokens, err := s.issueTokenPair(ctx, user.ID, user.Username, string(user.Role))
	if err != nil {
		return nil, apperror.Internal("刷新Token失败")
	}
	return tokens, nil
}

func (s *IdentityService) issueTokenPair(ctx context.Context, userID, username, role string) (*AuthTokens, error) {
	accessToken, err := auth.GenerateToken(s.jwtSecret, userID, username, role, auth.TokenTypeAccess, s.accessExpires, "")
	if err != nil {
		return nil, err
	}

	refreshTokenID := uuid.NewString()
	refreshToken, err := auth.GenerateToken(s.jwtSecret, userID, username, role, auth.TokenTypeRefresh, s.refreshExpires, refreshTokenID)
	if err != nil {
		return nil, err
	}

	if err := s.refreshStore.Save(ctx, userID, refreshTokenID, s.refreshExpires); err != nil {
		return nil, err
	}

	return &AuthTokens{
		Token:            accessToken,
		ExpiresIn:        int(s.accessExpires.Seconds()),
		RefreshToken:     refreshToken,
		RefreshExpiresIn: int(s.refreshExpires.Seconds()),
	}, nil
}

func (s *IdentityService) BootstrapAdmin(ctx context.Context, username, password string) error {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return nil
	}
	if len(username) < 3 || len(username) > 50 || len(password) < 8 || len(password) > 128 {
		return errors.New("invalid bootstrap admin credentials")
	}

	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return err
		}

		hashed, hashErr := auth.HashPassword(password)
		if hashErr != nil {
			return hashErr
		}

		return s.users.Create(ctx, &domain.User{
			Username:  username,
			Password:  hashed,
			CreatedAt: time.Now().UTC(),
			Role:      domain.UserRoleAdmin,
			Status:    domain.UserStatusActive,
		})
	}

	if user.Role != domain.UserRoleAdmin {
		if err := s.users.UpdateRole(ctx, user.ID, domain.UserRoleAdmin); err != nil {
			return err
		}
	}
	if user.Status != domain.UserStatusActive {
		if err := s.users.UpdateStatus(ctx, user.ID, domain.UserStatusActive); err != nil {
			return err
		}
	}
	return nil
}
