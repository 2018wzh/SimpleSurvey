package http

import (
	"net/http"
	"strings"

	"github.com/2018wzh/SimpleSurvey/backend/pkg/auth"
	"github.com/2018wzh/SimpleSurvey/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	ctxUserIDKey = "userID"
	ctxUserRole  = "userRole"
)

func (h *Handler) AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(c.GetHeader("Authorization"))
		if token == "" || !strings.HasPrefix(token, "Bearer ") {
			response.Error(c, http.StatusUnauthorized, 401, "Unauthorized", nil)
			c.Abort()
			return
		}
		token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
		claims, err := auth.ParseToken(jwtSecret, token)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, 401, "Token无效或已过期", nil)
			c.Abort()
			return
		}
		if claims.TokenType != "" && claims.TokenType != auth.TokenTypeAccess {
			response.Error(c, http.StatusUnauthorized, 401, "Token类型错误", nil)
			c.Abort()
			return
		}

		c.Set(ctxUserIDKey, claims.UserID)
		c.Set(ctxUserRole, claims.Role)
		c.Next()
	}
}

func (h *Handler) OptionalAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" {
			c.Next()
			return
		}
		if !strings.HasPrefix(header, "Bearer ") {
			response.Error(c, http.StatusUnauthorized, 401, "Authorization格式错误", nil)
			c.Abort()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		claims, err := auth.ParseToken(jwtSecret, token)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, 401, "Token无效或已过期", nil)
			c.Abort()
			return
		}
		if claims.TokenType != "" && claims.TokenType != auth.TokenTypeAccess {
			response.Error(c, http.StatusUnauthorized, 401, "Token类型错误", nil)
			c.Abort()
			return
		}
		c.Set(ctxUserIDKey, claims.UserID)
		c.Set(ctxUserRole, claims.Role)
		c.Next()
	}
}

func (h *Handler) AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, ok := c.Get(ctxUserRole)
		if !ok {
			response.Error(c, http.StatusForbidden, 403, "无管理员权限", nil)
			c.Abort()
			return
		}
		role, ok := roleValue.(string)
		if !ok || strings.TrimSpace(role) != "admin" {
			response.Error(c, http.StatusForbidden, 403, "无管理员权限", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

func getRequiredUserID(c *gin.Context) string {
	if value, ok := c.Get(ctxUserIDKey); ok {
		if userID, ok := value.(string); ok {
			return userID
		}
	}
	return ""
}

func getOptionalUserID(c *gin.Context) *string {
	if value, ok := c.Get(ctxUserIDKey); ok {
		if userID, ok := value.(string); ok && userID != "" {
			return &userID
		}
	}
	return nil
}
