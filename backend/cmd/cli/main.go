package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"flag"

	"github.com/2018wzh/SimpleSurvey/backend/internal/app"
	"github.com/2018wzh/SimpleSurvey/backend/internal/config"
	"github.com/2018wzh/SimpleSurvey/backend/internal/migration"
	mongoDriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd := flag.NewFlagSet("run", flag.ExitOnError)
		_ = runCmd.Parse(os.Args[2:])
		if err := app.RunServer(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "migrate":
		migrateCmd := flag.NewFlagSet("migrate", flag.ExitOnError)
		dryRun := migrateCmd.Bool("dry-run", false, "预演模式，不落库")
		timeout := migrateCmd.Duration("timeout", 30*time.Minute, "迁移超时时间")
		from := migrateCmd.String("from", "1.0", "迁移源版本，当前仅支持1.0")
		_ = migrateCmd.Parse(os.Args[2:])

		if err := migrateQuestionVersionID(*from, *dryRun, *timeout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "不支持的子命令: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("用法:")
	fmt.Println("  go run ./cmd/cli run")
	fmt.Println("  go run ./cmd/cli migrate [--from=1.0] [--dry-run] [--timeout=30m]")
}

func migrateQuestionVersionID(from string, dryRun bool, timeout time.Duration) error {
	if strings.TrimSpace(from) != "1.0" {
		return fmt.Errorf("当前仅支持从1.0迁移")
	}

	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	client, err := mongoDriver.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return fmt.Errorf("connect mongodb failed: %w", err)
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	db := client.Database(cfg.MongoDatabase)
	migrator := migration.NewQuestionVersionMigrator(db, time.Duration(cfg.RequestTimeoutSec)*time.Second)

	result, err := migrator.Migrate(ctx, dryRun)
	if err != nil {
		return fmt.Errorf("迁移失败: %w", err)
	}

	mode := "执行模式"
	if dryRun {
		mode = "预演模式"
	}

	fmt.Printf("v%s -> 新模型迁移完成（%s）\n", from, mode)
	fmt.Printf("questionnaires scanned=%d updated=%d questions patched=%d\n", result.QuestionnairesScanned, result.QuestionnairesUpdated, result.QuestionsPatched)
	fmt.Printf("responses scanned=%d updated=%d answers patched=%d\n", result.ResponsesScanned, result.ResponsesUpdated, result.AnswersPatched)
	fmt.Printf("questions generated=%d versions generated=%d\n", result.QuestionsGenerated, result.VersionsGenerated)
	return nil
}
