package telegram

import (
	"fmt"
	"strings"
	"time"

	"github.com/remnawave/limiter/internal/api"
)

// FormatManualAlert formats a manual alert message in HTML for Telegram.
func FormatManualAlert(user *api.CachedUser, ips []api.ActiveIP, limit int, loc *time.Location) string {
	var b strings.Builder

	b.WriteString("⚠️ <b>Превышение лимита устройств</b>\n\n")
	b.WriteString(fmt.Sprintf("👤 Пользователь: <code>%s</code>\n", escapeHTML(user.Username)))
	b.WriteString(fmt.Sprintf("📊 Лимит: %d | Обнаружено: %d IP\n", limit, len(ips)))
	b.WriteString(fmt.Sprintf("🕐 %s\n", time.Now().In(loc).Format("02.01.2006 15:04:05")))

	b.WriteString("\n📍 IP-адреса:\n")
	for _, ip := range ips {
		b.WriteString(fmt.Sprintf("  • <a href=\"https://ipinfo.io/%s\">%s</a> (нода: %s)\n", ip.IP, escapeHTML(ip.IP), escapeHTML(ip.NodeName)))
	}

	if user.SubscriptionURL != "" {
		b.WriteString(fmt.Sprintf("\n🔗 <a href=\"%s\">Профиль</a>", escapeHTML(user.SubscriptionURL)))
	}

	return b.String()
}

// FormatAutoAlert formats an auto-disable alert message in HTML for Telegram.
func FormatAutoAlert(user *api.CachedUser, ips []api.ActiveIP, limit int, durationMinutes int, loc *time.Location) string {
	var b strings.Builder

	b.WriteString("🔒 <b>Подписка автоматически отключена</b>\n\n")
	b.WriteString(fmt.Sprintf("👤 Пользователь: <code>%s</code>\n", escapeHTML(user.Username)))
	b.WriteString(fmt.Sprintf("📊 Лимит: %d | Обнаружено: %d IP\n", limit, len(ips)))

	if durationMinutes == 0 {
		b.WriteString("⏱ Отключена на: Перманентно\n")
	} else {
		b.WriteString(fmt.Sprintf("⏱ Отключена на: %d мин\n", durationMinutes))
	}

	b.WriteString(fmt.Sprintf("🕐 %s\n", time.Now().In(loc).Format("02.01.2006 15:04:05")))

	b.WriteString("\n📍 IP-адреса:\n")
	for _, ip := range ips {
		b.WriteString(fmt.Sprintf("  • <a href=\"https://ipinfo.io/%s\">%s</a> (нода: %s)\n", ip.IP, escapeHTML(ip.IP), escapeHTML(ip.NodeName)))
	}

	return b.String()
}

// FormatActionResult formats the result of an admin action.
func FormatActionResult(action, adminName, username string) string {
	var msg string
	switch action {
	case "drop":
		msg = "✅ Подключения сброшены"
	case "disable":
		msg = "🔒 Подписка отключена"
	case "ignore":
		msg = "🔇 Добавлен в whitelist"
	case "enable":
		msg = "🔓 Подписка включена"
	default:
		msg = "❓ Неизвестное действие"
	}
	return fmt.Sprintf("\n\n%s (админ: %s)", msg, escapeHTML(adminName))
}

// escapeHTML escapes special HTML characters for Telegram HTML parse mode.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
