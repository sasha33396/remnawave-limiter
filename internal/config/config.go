package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	RemnawaveAPIURL     string
	RemnawaveAPIToken   string
	CheckInterval       int
	ActiveIPWindow      int
	Tolerance           int
	ToleranceMultiplier float64
	Cooldown            int
	UserCacheTTL        int
	DefaultDeviceLimit  int
	ActionMode          string
	AutoDisableDuration int
	TelegramBotToken    string
	TelegramChatID      int64
	TelegramThreadID    int64
	TelegramAdminIDs    []int64
	WhitelistUserIDs    []string
	RedisURL            string
	Timezone            string
	Language            string
	RemnawaveCookies    string
	WebhookURL          string
	WebhookSecret       string
	SubnetGrouping         bool
	ViolationThreshold     int
	ViolationThresholdWindow int
}

func LoadConfig(envPath string) (*Config, error) {
	if envPath == "" {
		envPath = ".env"
	}

	if err := godotenv.Load(envPath); err != nil {
		logrus.Debug("Файл .env не найден, используются переменные окружения")
	}

	remnawaveAPIURL := os.Getenv("REMNAWAVE_API_URL")
	if remnawaveAPIURL == "" {
		return nil, fmt.Errorf("REMNAWAVE_API_URL обязательный параметр")
	}

	remnawaveAPIToken := os.Getenv("REMNAWAVE_API_TOKEN")
	if remnawaveAPIToken == "" {
		return nil, fmt.Errorf("REMNAWAVE_API_TOKEN обязательный параметр")
	}

	telegramBotToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramBotToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN обязательный параметр")
	}

	telegramChatIDStr := os.Getenv("TELEGRAM_CHAT_ID")
	if telegramChatIDStr == "" {
		return nil, fmt.Errorf("TELEGRAM_CHAT_ID обязательный параметр")
	}
	telegramChatID, err := strconv.ParseInt(telegramChatIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("TELEGRAM_CHAT_ID должен быть числом: %v", err)
	}

	telegramAdminIDsStr := os.Getenv("TELEGRAM_ADMIN_IDS")
	if telegramAdminIDsStr == "" {
		return nil, fmt.Errorf("TELEGRAM_ADMIN_IDS обязательный параметр")
	}
	telegramAdminIDs, err := parseint64list(telegramAdminIDsStr)
	if err != nil {
		return nil, fmt.Errorf("TELEGRAM_ADMIN_IDS: %v", err)
	}

	telegramThreadID := getEnvInt64("TELEGRAM_THREAD_ID", 0)

	actionMode := getEnv("ACTION_MODE", "manual")

	cfg := &Config{
		RemnawaveAPIURL:     remnawaveAPIURL,
		RemnawaveAPIToken:   remnawaveAPIToken,
		CheckInterval:       getEnvInt("CHECK_INTERVAL", 30),
		ActiveIPWindow:      getEnvInt("ACTIVE_IP_WINDOW", 300),
		Tolerance:           getEnvInt("TOLERANCE", 0),
		ToleranceMultiplier: getEnvFloat64("TOLERANCE_MULTIPLIER", 0),
		Cooldown:            getEnvInt("COOLDOWN", 300),
		UserCacheTTL:        getEnvInt("USER_CACHE_TTL", 600),
		DefaultDeviceLimit:  getEnvInt("DEFAULT_DEVICE_LIMIT", 0),
		ActionMode:          actionMode,
		AutoDisableDuration: getEnvInt("AUTO_DISABLE_DURATION", 0),
		TelegramBotToken:    telegramBotToken,
		TelegramChatID:      telegramChatID,
		TelegramThreadID:    telegramThreadID,
		TelegramAdminIDs:    telegramAdminIDs,
		WhitelistUserIDs:    parseList(getEnv("WHITELIST_USER_IDS", "")),
		RedisURL:            getEnv("REDIS_URL", "redis://redis:6379"),
		Timezone:            getEnv("TIMEZONE", "UTC"),
		Language:            getEnv("LANGUAGE", "ru"),
		RemnawaveCookies:    getEnv("REMNAWAVE_COOKIES", ""),
		WebhookURL:          getEnv("WEBHOOK_URL", ""),
		WebhookSecret:       getEnv("WEBHOOK_SECRET", ""),
		SubnetGrouping:         getEnvBool("SUBNET_GROUPING", false),
		ViolationThreshold:     getEnvInt("VIOLATION_THRESHOLD", 1),
		ViolationThresholdWindow: getEnvInt("VIOLATION_THRESHOLD_WINDOW", 3600),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (cfg *Config) Validate() error {
	if cfg.ActionMode != "manual" && cfg.ActionMode != "auto" {
		return fmt.Errorf("ACTION_MODE должен быть \"manual\" или \"auto\", получено %q", cfg.ActionMode)
	}
	if cfg.CheckInterval <= 0 {
		return fmt.Errorf("CHECK_INTERVAL должен быть > 0, получено %d", cfg.CheckInterval)
	}
	if cfg.ActiveIPWindow <= 0 {
		return fmt.Errorf("ACTIVE_IP_WINDOW должен быть > 0, получено %d", cfg.ActiveIPWindow)
	}
	if cfg.Cooldown <= 0 {
		return fmt.Errorf("COOLDOWN должен быть > 0, получено %d", cfg.Cooldown)
	}
	if cfg.ViolationThreshold <= 0 {
		return fmt.Errorf("VIOLATION_THRESHOLD должен быть > 0, получено %d", cfg.ViolationThreshold)
	}
	if cfg.ViolationThresholdWindow <= 0 {
		return fmt.Errorf("VIOLATION_THRESHOLD_WINDOW должен быть > 0, получено %d", cfg.ViolationThresholdWindow)
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		intVal, err := strconv.Atoi(value)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"key":     key,
				"value":   value,
				"default": defaultValue,
			}).Warnf("Не удалось преобразовать %s в число, используется значение по умолчанию %d", key, defaultValue)
			return defaultValue
		}
		return intVal
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"key":     key,
				"value":   value,
				"default": defaultValue,
			}).Warnf("Не удалось преобразовать %s в число, используется значение по умолчанию %d", key, defaultValue)
			return defaultValue
		}
		return intVal
	}
	return defaultValue
}

func getEnvFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"key":     key,
				"value":   value,
				"default": defaultValue,
			}).Warnf("Не удалось преобразовать %s в число, используется значение по умолчанию %v", key, defaultValue)
			return defaultValue
		}
		return floatVal
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.EqualFold(value, "true") || value == "1"
	}
	return defaultValue
}

func parseint64list(s string) ([]int64, error) {
	if s == "" {
		return []int64{}, nil
	}
	parts := strings.Split(s, ",")
	result := make([]int64, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		val, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("невозможно преобразовать %q в число: %v", trimmed, err)
		}
		result = append(result, val)
	}
	return result, nil
}

func parseList(listStr string) []string {
	if listStr == "" {
		return []string{}
	}
	items := strings.Split(listStr, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
