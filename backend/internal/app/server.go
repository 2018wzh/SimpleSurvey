package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/config"
	httpDelivery "github.com/2018wzh/SimpleSurvey/backend/internal/delivery/http"
	"github.com/2018wzh/SimpleSurvey/backend/internal/migration"
	"github.com/2018wzh/SimpleSurvey/backend/internal/repository/mongo"
	redisrepo "github.com/2018wzh/SimpleSurvey/backend/internal/repository/redis"
	"github.com/2018wzh/SimpleSurvey/backend/internal/service"
	"github.com/2018wzh/SimpleSurvey/backend/pkg/logger"
	goredis "github.com/redis/go-redis/v9"
	mongoDriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func RunServer() error {
	cfg := config.Load()

	log, err := logger.New(cfg.AppEnv)
	if err != nil {
		return fmt.Errorf("init logger failed: %w", err)
	}
	defer func() {
		_ = log.Sync()
	}()

	ctx := context.Background()
	mongoClient, err := mongoDriver.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return fmt.Errorf("connect mongodb failed: %w", err)
	}
	defer func() {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mongoClient.Disconnect(disconnectCtx)
	}()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := mongoClient.Ping(pingCtx, nil); err != nil {
		pingCancel()
		return fmt.Errorf("ping mongodb failed: %w", err)
	}
	pingCancel()

	db := mongoClient.Database(cfg.MongoDatabase)
	timeout := time.Duration(cfg.RequestTimeoutSec) * time.Second

	startupMigrationCtx, startupMigrationCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	startupValidation, err := migration.EnsureSchemaAtStartup(startupMigrationCtx, db, timeout)
	startupMigrationCancel()
	if err != nil {
		return fmt.Errorf("startup database schema validation failed: %w", err)
	}
	if startupValidation.Migrated {
		log.Info("detected v1 data and migration executed",
			zap.Int("questionnairesScanned", startupValidation.Migration.QuestionnairesScanned),
			zap.Int("questionnairesUpdated", startupValidation.Migration.QuestionnairesUpdated),
			zap.Int("responsesScanned", startupValidation.Migration.ResponsesScanned),
			zap.Int("responsesUpdated", startupValidation.Migration.ResponsesUpdated),
		)
	}

	redisClient := goredis.NewClient(&goredis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer func() {
		_ = redisClient.Close()
	}()

	redisPingCtx, redisPingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := redisClient.Ping(redisPingCtx).Err(); err != nil {
		redisPingCancel()
		return fmt.Errorf("ping redis failed: %w", err)
	}
	redisPingCancel()

	userRepo := mongo.NewUserRepository(db, timeout)
	questionnaireRepo := mongo.NewQuestionnaireRepository(db, timeout)
	responseRepo := mongo.NewResponseRepository(db, timeout)
	questionRepo := mongo.NewQuestionRepository(db, timeout)
	questionBankRepo := mongo.NewQuestionBankRepository(db, timeout)
	refreshTokenStore := redisrepo.NewRefreshTokenStore(redisClient, cfg.RedisKeyPrefix, timeout)

	indexCtx, indexCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := userRepo.EnsureIndexes(indexCtx); err != nil {
		indexCancel()
		return fmt.Errorf("create user indexes failed: %w", err)
	}
	if err := questionnaireRepo.EnsureIndexes(indexCtx); err != nil {
		indexCancel()
		return fmt.Errorf("create questionnaire indexes failed: %w", err)
	}
	if err := responseRepo.EnsureIndexes(indexCtx); err != nil {
		indexCancel()
		return fmt.Errorf("create response indexes failed: %w", err)
	}
	if err := questionRepo.EnsureIndexes(indexCtx); err != nil {
		indexCancel()
		return fmt.Errorf("create question indexes failed: %w", err)
	}
	if err := questionBankRepo.EnsureIndexes(indexCtx); err != nil {
		indexCancel()
		return fmt.Errorf("create question bank indexes failed: %w", err)
	}
	indexCancel()

	identityService := service.NewIdentityService(
		userRepo,
		refreshTokenStore,
		cfg.JWTSecret,
		time.Duration(cfg.AccessTokenExpiresHours)*time.Hour,
		time.Duration(cfg.RefreshTokenExpiresHours)*time.Hour,
	)
	questionnaireService := service.NewQuestionnaireService(questionnaireRepo, responseRepo)
	adminService := service.NewAdminService(userRepo, questionnaireRepo)
	questionService := service.NewQuestionService(questionRepo, questionnaireRepo, responseRepo)
	questionBankService := service.NewQuestionBankService(questionBankRepo)

	bootstrapCtx, bootstrapCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := identityService.BootstrapAdmin(bootstrapCtx, cfg.AdminBootstrapUsername, cfg.AdminBootstrapPassword); err != nil {
		bootstrapCancel()
		return fmt.Errorf("bootstrap admin failed: %w", err)
	}
	bootstrapCancel()

	handler := httpDelivery.NewHandler(identityService, questionnaireService, adminService, questionService, questionBankService)
	router := httpDelivery.NewRouter(cfg, handler, log)

	srv := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           router,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("server started", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("server shutdown failed", zap.Error(err))
			return fmt.Errorf("server shutdown failed: %w", err)
		}
		log.Info("server shutdown completed")
		return nil
	case err := <-errCh:
		log.Error("server crashed", zap.Error(err))
		return fmt.Errorf("server crashed: %w", err)
	}
}
