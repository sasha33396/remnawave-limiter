package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/remnawave/limiter/internal/api"
	"github.com/remnawave/limiter/internal/cache"
	"github.com/remnawave/limiter/internal/config"
	"github.com/remnawave/limiter/internal/monitor"
	"github.com/remnawave/limiter/internal/telegram"
	"github.com/remnawave/limiter/internal/version"
)

func main() {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	logger.Infof("Remnawave Limiter v%s", version.Version)

	cfg, err := config.LoadConfig("")
	if err != nil {
		logger.Fatalf("Ошибка конфигурации: %v", err)
	}

	logger.Infof("Режим: %s", cfg.ActionMode)
	logger.Infof("Интервал проверки: %dс", cfg.CheckInterval)
	logger.Infof("API: %s", cfg.RemnawaveAPIURL)

	// Init Redis
	redisCache, err := cache.New(cfg.RedisURL)
	if err != nil {
		logger.Fatalf("Ошибка Redis: %v", err)
	}
	defer redisCache.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := redisCache.Ping(ctx); err != nil {
		logger.Fatalf("Redis недоступен: %v", err)
	}
	logger.Info("Redis подключён")

	redisCache.InitWhitelist(ctx, cfg.WhitelistUserIDs)

	// Init API client
	apiClient := api.NewClient(cfg.RemnawaveAPIURL, cfg.RemnawaveAPIToken)
	apiClient.SetLogger(logger)

	// Init Telegram bot
	bot, err := telegram.NewBot(cfg.TelegramBotToken, cfg.TelegramChatID, cfg.TelegramThreadID, cfg.TelegramAdminIDs, logger)
	if err != nil {
		logger.Fatalf("Ошибка Telegram: %v", err)
	}
	logger.Info("Telegram бот подключён")

	// Init monitor
	mon, err := monitor.New(cfg, apiClient, redisCache, bot, logger)
	if err != nil {
		logger.Fatalf("Ошибка монитора: %v", err)
	}

	// Wire callback handler
	bot.SetActionHandler(func(ctx context.Context, action, userUUID, userID string) error {
		switch action {
		case "drop":
			return apiClient.DropConnections(ctx, []string{userUUID})
		case "disable":
			return apiClient.DisableUser(ctx, userUUID)
		case "enable":
			return apiClient.EnableUser(ctx, userUUID)
		case "ignore":
			return redisCache.AddToWhitelist(ctx, userID)
		}
		return nil
	})

	// Handle shutdown
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go bot.StartPolling(sigCtx)
	mon.Run(sigCtx)

	logger.Info("Remnawave Limiter остановлен")
}
