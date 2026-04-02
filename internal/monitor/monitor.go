package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/remnawave/limiter/internal/api"
	"github.com/remnawave/limiter/internal/cache"
	"github.com/remnawave/limiter/internal/config"
	"github.com/remnawave/limiter/internal/telegram"
)

// Monitor is the core monitoring loop that polls active nodes for user IPs
// and detects violations when a subscription is used from too many devices.
type Monitor struct {
	config   *config.Config
	api      *api.Client
	cache    *cache.Cache
	bot      *telegram.Bot
	logger   *logrus.Logger
	location *time.Location
}

// New creates a new Monitor with the given dependencies. It loads the timezone
// from the config and returns an error if the timezone is invalid.
func New(cfg *config.Config, apiClient *api.Client, c *cache.Cache, bot *telegram.Bot, logger *logrus.Logger) (*Monitor, error) {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("неверная таймзона %q: %w", cfg.Timezone, err)
	}

	return &Monitor{
		config:   cfg,
		api:      apiClient,
		cache:    c,
		bot:      bot,
		logger:   logger,
		location: loc,
	}, nil
}

// Run starts the main monitoring loop. It ticks at CheckInterval and, if
// auto mode with a duration is configured, also runs the restore loop.
func (m *Monitor) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(m.config.CheckInterval) * time.Second)
	defer ticker.Stop()

	if m.config.ActionMode == "auto" && m.config.AutoDisableDuration > 0 {
		go m.restoreLoop(ctx)
	}

	m.logger.Info("🚀 Мониторинг запущен")

	// Run an initial check immediately
	m.check(ctx)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Мониторинг остановлен")
			return
		case <-ticker.C:
			m.check(ctx)
		}
	}
}

// check performs a single monitoring cycle: fetches active nodes, collects
// user IPs from all nodes in parallel, aggregates them, and checks each user.
func (m *Monitor) check(ctx context.Context) {
	nodes, err := m.api.GetActiveNodes(ctx)
	if err != nil {
		m.logger.WithError(err).Error("Ошибка получения активных нод")
		return
	}

	if len(nodes) == 0 {
		m.logger.Debug("Нет активных нод")
		return
	}

	// Fetch user IPs from all nodes in parallel
	type nodeResult struct {
		nodeName string
		nodeUUID string
		entries  []api.UserIPEntry
		err      error
	}

	results := make([]nodeResult, len(nodes))
	var wg sync.WaitGroup

	for i, node := range nodes {
		wg.Add(1)
		go func(idx int, n api.Node) {
			defer wg.Done()
			entries, err := m.api.FetchUsersIPs(ctx, n.UUID)
			results[idx] = nodeResult{
				nodeName: n.Name,
				nodeUUID: n.UUID,
				entries:  entries,
				err:      err,
			}
		}(i, node)
	}

	wg.Wait()

	// Aggregate: map[userID] → []ActiveIP
	activeWindow := time.Duration(m.config.ActiveIPWindow) * time.Second
	cutoff := time.Now().Add(-activeWindow)
	aggregated := make(map[string][]api.ActiveIP)

	for _, res := range results {
		if res.err != nil {
			m.logger.WithError(res.err).WithField("node", res.nodeName).Error("Ошибка получения IP с ноды")
			continue
		}

		for _, entry := range res.entries {
			for _, ip := range entry.IPs {
				if ip.LastSeen.Before(cutoff) {
					continue
				}
				aggregated[entry.UserID] = append(aggregated[entry.UserID], api.ActiveIP{
					IP:       ip.IP,
					LastSeen: ip.LastSeen,
					NodeName: res.nodeName,
					NodeUUID: res.nodeUUID,
				})
			}
		}
	}

	m.logger.WithField("users", len(aggregated)).Debug("Проверка пользователей")

	for userID, ips := range aggregated {
		m.checkUser(ctx, userID, ips)
	}
}

// checkUser evaluates a single user's active IPs against their device limit.
func (m *Monitor) checkUser(ctx context.Context, userID string, activeIPs []api.ActiveIP) {
	// Deduplicate IPs by address (keep latest lastSeen)
	uniqueMap := make(map[string]api.ActiveIP)
	for _, ip := range activeIPs {
		existing, ok := uniqueMap[ip.IP]
		if !ok || ip.LastSeen.After(existing.LastSeen) {
			uniqueMap[ip.IP] = ip
		}
	}

	uniqueIPs := make([]api.ActiveIP, 0, len(uniqueMap))
	for _, ip := range uniqueMap {
		uniqueIPs = append(uniqueIPs, ip)
	}

	// Check whitelist
	whitelisted, err := m.cache.IsWhitelisted(ctx, userID)
	if err != nil {
		m.logger.WithError(err).WithField("userID", userID).Error("Ошибка проверки whitelist")
		return
	}
	if whitelisted {
		return
	}

	// Get user data
	user, err := m.getUser(ctx, userID)
	if err != nil {
		m.logger.WithError(err).WithField("userID", userID).Error("Ошибка получения данных пользователя")
		return
	}

	// Resolve device limit
	limit := m.resolveLimit(user.HWIDDeviceLimit)
	if limit == 0 {
		// Unlimited
		return
	}

	// Check if violation
	if len(uniqueIPs) <= limit+m.config.Tolerance {
		return
	}

	// Check cooldown
	active, err := m.cache.IsCooldownActive(ctx, userID)
	if err != nil {
		m.logger.WithError(err).WithField("userID", userID).Error("Ошибка проверки cooldown")
		return
	}
	if active {
		return
	}

	// Set cooldown
	if err := m.cache.SetCooldown(ctx, userID, time.Duration(m.config.Cooldown)*time.Second); err != nil {
		m.logger.WithError(err).WithField("userID", userID).Error("Ошибка установки cooldown")
	}

	m.logger.WithFields(logrus.Fields{
		"userID":   userID,
		"username": user.Username,
		"ips":      len(uniqueIPs),
		"limit":    limit,
	}).Warn("Обнаружено превышение лимита устройств")

	if m.config.ActionMode == "auto" {
		m.handleAutoAction(ctx, user, uniqueIPs, limit)
	} else {
		m.handleManualAction(user, uniqueIPs, limit)
	}
}

// getUser retrieves a user from cache, falling back to an API call on miss.
func (m *Monitor) getUser(ctx context.Context, userID string) (*api.CachedUser, error) {
	cached, err := m.cache.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("cache get user: %w", err)
	}
	if cached != nil {
		return cached, nil
	}

	// Cache miss — fetch from API
	userData, err := m.api.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("api get user: %w", err)
	}

	// Convert to CachedUser
	cu := &api.CachedUser{
		UUID:     userData.UUID,
		UserID:   userID,
		Username: userData.Username,
		Status:   userData.Status,
	}

	if userData.Email != nil {
		cu.Email = *userData.Email
	}
	if userData.TelegramID != nil {
		cu.TelegramID = *userData.TelegramID
	}
	if userData.HWIDDeviceLimit != nil {
		cu.HWIDDeviceLimit = *userData.HWIDDeviceLimit
	} else {
		cu.HWIDDeviceLimit = -1 // null → use default
	}
	cu.SubscriptionURL = userData.SubscriptionURL

	// Cache it
	ttl := time.Duration(m.config.UserCacheTTL) * time.Second
	if err := m.cache.SetUser(ctx, userID, cu, ttl); err != nil {
		m.logger.WithError(err).WithField("userID", userID).Warn("Ошибка кэширования пользователя")
	}

	return cu, nil
}

// resolveLimit returns the effective device limit for a user.
// 0 means unlimited (skip check), -1 means use the default from config.
func (m *Monitor) resolveLimit(hwidDeviceLimit int) int {
	if hwidDeviceLimit == 0 {
		return 0 // unlimited
	}
	if hwidDeviceLimit == -1 {
		return m.config.DefaultDeviceLimit
	}
	return hwidDeviceLimit
}

// handleManualAction sends a manual alert via Telegram with action buttons.
func (m *Monitor) handleManualAction(user *api.CachedUser, ips []api.ActiveIP, limit int) {
	text := telegram.FormatManualAlert(user, ips, limit, m.location)
	if err := m.bot.SendManualAlert(text, user.UUID, user.UserID); err != nil {
		m.logger.WithError(err).WithField("userID", user.UserID).Error("Ошибка отправки manual alert")
	}
}

// handleAutoAction disables the user, optionally sets a restore timer,
// and sends an auto alert via Telegram.
func (m *Monitor) handleAutoAction(ctx context.Context, user *api.CachedUser, ips []api.ActiveIP, limit int) {
	if err := m.api.DisableUser(ctx, user.UUID); err != nil {
		m.logger.WithError(err).WithField("userID", user.UserID).Error("Ошибка отключения пользователя")
		return
	}

	if m.config.AutoDisableDuration > 0 {
		duration := time.Duration(m.config.AutoDisableDuration) * time.Minute
		if err := m.cache.SetRestoreTimer(ctx, user.UUID, duration); err != nil {
			m.logger.WithError(err).WithField("userID", user.UserID).Error("Ошибка установки таймера восстановления")
		}
	}

	text := telegram.FormatAutoAlert(user, ips, limit, m.config.AutoDisableDuration, m.location)
	if err := m.bot.SendAutoAlert(text, user.UUID); err != nil {
		m.logger.WithError(err).WithField("userID", user.UserID).Error("Ошибка отправки auto alert")
	}
}

// restoreLoop periodically checks for expired restore timers and re-enables
// the corresponding users.
func (m *Monitor) restoreLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expired, err := m.cache.GetExpiredRestoreTimers(ctx)
			if err != nil {
				m.logger.WithError(err).Error("Ошибка получения истёкших таймеров восстановления")
				continue
			}

			for _, uuid := range expired {
				if err := m.api.EnableUser(ctx, uuid); err != nil {
					m.logger.WithError(err).WithField("uuid", uuid).Error("Ошибка включения пользователя по таймеру")
					continue
				}

				m.logger.WithField("uuid", uuid).Info("Пользователь автоматически включён по таймеру")

				msg := fmt.Sprintf("🔓 Подписка <code>%s</code> автоматически включена по таймеру", uuid)
				if err := m.bot.SendMessage(msg); err != nil {
					m.logger.WithError(err).Error("Ошибка отправки уведомления о восстановлении")
				}
			}
		}
	}
}
