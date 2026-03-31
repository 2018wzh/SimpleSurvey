package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/config"
	"github.com/2018wzh/SimpleSurvey/backend/internal/repository/mongo"
	"github.com/2018wzh/SimpleSurvey/backend/internal/service"
	mongoDriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/term"
)

func main() {
	username := flag.String("username", "admin", "管理员用户名")
	password := flag.String("password", "", "新密码（若为空则交互式输入）")
	flag.Parse()

	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongoDriver.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect mongodb failed: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	db := client.Database(cfg.MongoDatabase)
	userRepo := mongo.NewUserRepository(db, time.Duration(cfg.RequestTimeoutSec)*time.Second)

	svc := service.NewIdentityService(userRepo, nil, cfg.JWTSecret, time.Duration(cfg.AccessTokenExpiresHours)*time.Hour, time.Duration(cfg.RefreshTokenExpiresHours)*time.Hour)

	newPass := strings.TrimSpace(*password)
	if newPass == "" {
		fmt.Print("输入新密码: ")
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "读取密码失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Print("重复新密码: ")
		b2, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "读取密码失败: %v\n", err)
			os.Exit(1)
		}
		if string(b) != string(b2) {
			fmt.Fprintln(os.Stderr, "两次输入的密码不一致")
			os.Exit(1)
		}
		newPass = string(b)
	}

	if err := svc.ResetAdminPassword(context.Background(), *username, newPass); err != nil {
		fmt.Fprintf(os.Stderr, "重置密码失败: %s\n", err.Message)
		os.Exit(1)
	}
	fmt.Println("管理员密码已重置成功")
}
