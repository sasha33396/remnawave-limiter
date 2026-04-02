package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// ActionHandler is called when an admin presses an inline button.
// action is one of: "drop", "disable", "ignore", "enable".
type ActionHandler func(ctx context.Context, action, userUUID, userID string) error

// Bot wraps the Telegram Bot API for sending alerts and handling callbacks.
type Bot struct {
	api      *tgbotapi.BotAPI
	chatID   int64
	threadID int64
	adminIDs map[int64]bool
	logger   *logrus.Logger
	onAction ActionHandler
}

// NewBot creates a new Bot instance with the given configuration.
func NewBot(token string, chatID, threadID int64, adminIDs []int64, logger *logrus.Logger) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать Telegram бота: %w", err)
	}

	admins := make(map[int64]bool, len(adminIDs))
	for _, id := range adminIDs {
		admins[id] = true
	}

	return &Bot{
		api:      botAPI,
		chatID:   chatID,
		threadID: threadID,
		adminIDs: admins,
		logger:   logger,
	}, nil
}

// SetActionHandler sets the callback handler for inline button presses.
func (b *Bot) SetActionHandler(handler ActionHandler) {
	b.onAction = handler
}

// sendMsg sends a message with optional inline keyboard and thread support.
func (b *Bot) sendMsg(text string, keyboard *tgbotapi.InlineKeyboardMarkup) error {
	params := tgbotapi.Params{}
	params.AddFirstValid("chat_id", b.chatID)
	params["text"] = text
	params["parse_mode"] = tgbotapi.ModeHTML
	params["disable_web_page_preview"] = "true"

	if b.threadID != 0 {
		params.AddNonZero64("message_thread_id", b.threadID)
	}

	if keyboard != nil {
		data, err := json.Marshal(keyboard)
		if err != nil {
			return fmt.Errorf("не удалось сериализовать клавиатуру: %w", err)
		}
		params["reply_markup"] = string(data)
	}

	_, err := b.api.MakeRequest("sendMessage", params)
	if err != nil {
		return fmt.Errorf("не удалось отправить сообщение: %w", err)
	}
	return nil
}

// SendManualAlert sends a manual alert with 3 action buttons.
func (b *Bot) SendManualAlert(text string, userUUID string, userID string) error {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Сбросить подключения", fmt.Sprintf("drop:%s:%s", userUUID, userID)),
			tgbotapi.NewInlineKeyboardButtonData("🔒 Отключить подписку", fmt.Sprintf("disable:%s:%s", userUUID, userID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔇 Игнорировать", fmt.Sprintf("ignore:%s:%s", userUUID, userID)),
		),
	)
	return b.sendMsg(text, &keyboard)
}

// SendAutoAlert sends an auto-disable alert with 1 "Enable" button.
func (b *Bot) SendAutoAlert(text string, userUUID string) error {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔓 Включить подписку", fmt.Sprintf("enable:%s:", userUUID)),
		),
	)
	return b.sendMsg(text, &keyboard)
}

// SendMessage sends a plain text message without any keyboard.
func (b *Bot) SendMessage(text string) error {
	return b.sendMsg(text, nil)
}

// StartPolling starts a long polling loop for handling inline button callbacks.
// It blocks until the context is cancelled.
func (b *Bot) StartPolling(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := b.api.GetUpdatesChan(u)

	b.logger.Info("Telegram бот: запущен polling")

	for {
		select {
		case <-ctx.Done():
			b.logger.Info("Telegram бот: polling остановлен")
			b.api.StopReceivingUpdates()
			return
		case update := <-updates:
			if update.CallbackQuery == nil {
				continue
			}
			b.handleCallback(ctx, update.CallbackQuery)
		}
	}
}

// handleCallback processes an inline button callback.
func (b *Bot) handleCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	callerID := callback.From.ID

	// Check admin access
	if !b.adminIDs[callerID] {
		answer := tgbotapi.NewCallback(callback.ID, "⛔ Нет доступа")
		if _, err := b.api.Request(answer); err != nil {
			b.logger.WithError(err).Error("Telegram бот: ошибка ответа на callback")
		}
		return
	}

	// Parse callback data: action:userUUID:userID
	parts := strings.SplitN(callback.Data, ":", 3)
	if len(parts) < 3 {
		b.logger.WithField("data", callback.Data).Warn("Telegram бот: неверный формат callback data")
		return
	}

	action := parts[0]
	userUUID := parts[1]
	userID := parts[2]

	adminName := callback.From.FirstName
	if callback.From.LastName != "" {
		adminName += " " + callback.From.LastName
	}
	if callback.From.UserName != "" {
		adminName = "@" + callback.From.UserName
	}

	// Call action handler
	if b.onAction != nil {
		if err := b.onAction(ctx, action, userUUID, userID); err != nil {
			b.logger.WithError(err).WithFields(logrus.Fields{
				"action":   action,
				"userUUID": userUUID,
				"userID":   userID,
			}).Error("Telegram бот: ошибка выполнения действия")

			answer := tgbotapi.NewCallback(callback.ID, "❌ Ошибка: "+err.Error())
			if _, err := b.api.Request(answer); err != nil {
				b.logger.WithError(err).Error("Telegram бот: ошибка ответа на callback")
			}
			return
		}
	}

	// Use userUUID as fallback for username display in action result
	username := userUUID

	// Edit original message to append action result and remove keyboard
	actionResult := FormatActionResult(action, adminName, username)

	originalHTML := callback.Message.Text
	if originalHTML == "" {
		originalHTML = callback.Message.Caption
	}

	editMsg := tgbotapi.NewEditMessageText(b.chatID, callback.Message.MessageID, originalHTML+actionResult)
	editMsg.ParseMode = tgbotapi.ModeHTML
	editMsg.DisableWebPagePreview = true
	// Remove keyboard by setting empty markup
	emptyMarkup := tgbotapi.NewInlineKeyboardMarkup()
	editMsg.ReplyMarkup = &emptyMarkup

	if _, err := b.api.Request(editMsg); err != nil {
		b.logger.WithError(err).Error("Telegram бот: ошибка редактирования сообщения")
	}

	// Send callback answer
	answer := tgbotapi.NewCallback(callback.ID, "✅ Выполнено")
	if _, err := b.api.Request(answer); err != nil {
		b.logger.WithError(err).Error("Telegram бот: ошибка ответа на callback")
	}
}
