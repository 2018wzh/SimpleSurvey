package http

import (
	"fmt"

	"github.com/2018wzh/SimpleSurvey/backend/internal/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewRouter(cfg config.Config, handler *Handler, log *zap.Logger) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-Id"},
		ExposeHeaders:    []string{"X-Request-Id"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", handler.Register)
			auth.POST("/login", handler.Login)
			auth.POST("/refresh", handler.RefreshToken)
		}

		questionnaires := api.Group("/questionnaires")
		questionnaires.Use(handler.AuthRequired(cfg.JWTSecret))
		{
			questionnaires.POST("", handler.CreateQuestionnaire)
			questionnaires.GET("", handler.GetQuestionnaires)
			questionnaires.PATCH("/:id/status", handler.UpdateQuestionnaireStatus)
			questionnaires.GET("/:id/stats", handler.GetQuestionnaireStats)
			questionnaires.GET("/:id/responses", handler.GetQuestionnaireResponses)
		}

		admin := api.Group("/admin")
		admin.Use(handler.AuthRequired(cfg.JWTSecret), handler.AdminRequired())
		{
			admin.GET("/users", handler.AdminListUsers)
			admin.PATCH("/users/:id/role", handler.AdminUpdateUserRole)
			admin.PATCH("/users/:id/status", handler.AdminUpdateUserStatus)

			admin.GET("/questionnaires", handler.AdminListQuestionnaires)
			admin.PATCH("/questionnaires/:id/status", handler.AdminUpdateQuestionnaireStatus)
		}

		surveys := api.Group("/surveys")
		surveys.Use(handler.OptionalAuth(cfg.JWTSecret))
		{
			surveys.GET("/:id", handler.GetSurvey)
			surveys.POST("/:id/responses", handler.SubmitResponse)
		}
	}

	log.Info(fmt.Sprintf("router initialized, env=%s", cfg.AppEnv))
	return r
}
