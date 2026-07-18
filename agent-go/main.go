package main

import (
	"agent-go/client"
	"agent-go/config"
	"agent-go/controller"
	esdao "agent-go/dao/elasticsearch"
	mysqldao "agent-go/dao/mysql"
	redisdao "agent-go/dao/redis"
	"agent-go/service"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

func main() {
	configPath := flag.String("config", "config.json", "configuration file")
	listenAddr := flag.String("listen", ":8080", "HTTP listen address")
	flag.Parse()
	configureLogging()
	slog.Info("service_starting", "service", "agent-go", "config", *configPath, "listen", *listenAddr)
	cfg, err := config.Load(*configPath)
	if err != nil {
		exit(err)
	}
	ctx := context.Background()
	db, err := mysqldao.Open(ctx, cfg.MySQL)
	if err != nil {
		exit(err)
	}
	defer db.Close()
	slog.Info("dependency_ready", "dependency", "mysql")
	esClient := esdao.NewClient(cfg.Elasticsearch)
	memoryDAO := esdao.NewMemoryDAO(esClient, cfg.Indices.ConversationMemory)
	if err := memoryDAO.EnsureIndex(ctx); err != nil {
		exit(fmt.Errorf("initialize Elasticsearch memory index: %w", err))
	}
	slog.Info("dependency_ready", "dependency", "elasticsearch", "index", cfg.Indices.ConversationMemory)
	stateDAO := redisdao.NewDialogueStateDAO(cfg.Redis)
	defer stateDAO.Close()
	if err := stateDAO.Ping(ctx); err != nil {
		exit(err)
	}
	slog.Info("dependency_ready", "dependency", "redis", "address", cfg.Redis.Address)
	python, err := client.NewPythonAgent(cfg.PythonAgent)
	if err != nil {
		exit(err)
	}
	defer python.Close()
	slog.Info("dependency_configured", "dependency", "python_grpc", "address", cfg.PythonAgent.Address)
	authService := service.NewAuthService(mysqldao.NewUserDAO(db), cfg.Auth)
	chatService := service.NewChatService(memoryDAO, mysqldao.NewSessionDAO(db), mysqldao.NewUsageDAO(db), stateDAO, python, cfg.Memory.RecallLimit)
	if err := controller.NewServer(authService, chatService, cfg.Auth).Run(*listenAddr); err != nil {
		exit(err)
	}
}

func configureLogging() {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
}

func exit(err error) {
	slog.Error("service_stopped", "service", "agent-go", "error", err)
	fmt.Fprintln(os.Stderr, "错误:", err)
	os.Exit(1)
}
