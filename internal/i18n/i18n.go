package i18n

var current = "ru"

var translations = map[string]map[string]string{
	"ru": {
		"alert.manual.title":    "⚠️ <b>Превышение лимита устройств</b>",
		"alert.auto.title":      "🔒 <b>Подписка автоматически отключена</b>",
		"alert.user":            "👤 Пользователь",
		"alert.limit":           "📊 Лимит",
		"alert.detected_ips":    "Обнаружено",
		"alert.subnets":         "Подсетей",
		"alert.violations_24h":  "📈 Нарушений за 24ч",
		"alert.disabled_for":    "⏱ Отключена на",
		"alert.permanent":       "Перманентно",
		"alert.ips_header":      "📍 IP-адреса",
		"alert.node":            "нода",
		"alert.and_more":        "и ещё",
		"alert.profile":         "🔗 Профиль",

		"action.drop":           "✅ Подключения сброшены",
		"action.disable":        "🔒 Подписка отключена навсегда",
		"action.disable_temp":   "🔒 Подписка временно отключена",
		"action.ignore":         "🔇 Добавлен в whitelist",
		"action.enable":         "🔓 Подписка включена",
		"action.unknown":        "❓ Неизвестное действие",
		"action.admin":          "админ",

		"button.drop":           "🔄 Сбросить подключения",
		"button.disable_forever": "🔒 Отключить навсегда",
		"button.disable_for":    "🔒 Отключить на",
		"button.ignore":         "🔇 Игнорировать",
		"button.enable":         "🔓 Включить подписку",

		"callback.no_access":    "⛔ Нет доступа",
		"callback.done":         "✅ Выполнено",
		"callback.error":        "❌ Ошибка",

		"restore.message":       "🔓 Подписка <code>%s</code> автоматически включена по таймеру",

		"duration.forever":      "навсегда",
		"duration.min":          "мин",
		"duration.hour":         "ч",
		"duration.day":          "д",

		"log.monitoring_started":    "🚀 Мониторинг запущен",
		"log.monitoring_stopped":    "Мониторинг остановлен",
		"log.limit_exceeded":        "Обнаружено превышение лимита устройств",
		"log.threshold_not_reached": "Порог нарушений не достигнут",

		"startup.title":           "🚀 <b>Remnawave Limiter запущен</b>",
		"startup.version":         "Версия",
		"startup.mode":            "Режим",
		"startup.mode_manual":     "Ручной",
		"startup.mode_auto":       "Автоматический",
		"startup.interval":        "Интервал проверки",
		"startup.cooldown":        "Кулдаун",
		"startup.tolerance":       "Допуск",
		"startup.tolerance_mult":  "Множитель допуска",
		"startup.default_limit":   "Лимит по умолчанию",
		"startup.unlimited":       "не ограничено",
		"startup.auto_disable":    "Авто-отключение",
		"startup.webhook":         "Webhook",
		"startup.enabled":         "включён",
		"startup.disabled":        "выключен",
		"startup.subnet_grouping":       "Группировка подсетей",
		"startup.violation_threshold":   "Порог нарушений",
		"startup.threshold_window":      "Окно порога",
		"startup.sec":                   "с",
	},
	"en": {
		"alert.manual.title":    "⚠️ <b>Device limit exceeded</b>",
		"alert.auto.title":      "🔒 <b>Subscription automatically disabled</b>",
		"alert.user":            "👤 User",
		"alert.limit":           "📊 Limit",
		"alert.detected_ips":    "Detected",
		"alert.subnets":         "Subnets",
		"alert.violations_24h":  "📈 Violations in 24h",
		"alert.disabled_for":    "⏱ Disabled for",
		"alert.permanent":       "Permanently",
		"alert.ips_header":      "📍 IP addresses",
		"alert.node":            "node",
		"alert.and_more":        "and more",
		"alert.profile":         "🔗 Profile",

		"action.drop":           "✅ Connections dropped",
		"action.disable":        "🔒 Subscription disabled permanently",
		"action.disable_temp":   "🔒 Subscription temporarily disabled",
		"action.ignore":         "🔇 Added to whitelist",
		"action.enable":         "🔓 Subscription enabled",
		"action.unknown":        "❓ Unknown action",
		"action.admin":          "admin",

		"button.drop":           "🔄 Drop connections",
		"button.disable_forever": "🔒 Disable permanently",
		"button.disable_for":    "🔒 Disable for",
		"button.ignore":         "🔇 Ignore",
		"button.enable":         "🔓 Enable subscription",

		"callback.no_access":    "⛔ Access denied",
		"callback.done":         "✅ Done",
		"callback.error":        "❌ Error",

		"restore.message":       "🔓 Subscription <code>%s</code> automatically enabled by timer",

		"duration.forever":      "forever",
		"duration.min":          "min",
		"duration.hour":         "h",
		"duration.day":          "d",

		"log.monitoring_started":    "🚀 Monitoring started",
		"log.monitoring_stopped":    "Monitoring stopped",
		"log.limit_exceeded":        "Device limit exceeded",
		"log.threshold_not_reached": "Violation threshold not reached",

		"startup.title":           "🚀 <b>Remnawave Limiter started</b>",
		"startup.version":         "Version",
		"startup.mode":            "Mode",
		"startup.mode_manual":     "Manual",
		"startup.mode_auto":       "Automatic",
		"startup.interval":        "Check interval",
		"startup.cooldown":        "Cooldown",
		"startup.tolerance":       "Tolerance",
		"startup.tolerance_mult":  "Tolerance multiplier",
		"startup.default_limit":   "Default limit",
		"startup.unlimited":       "unlimited",
		"startup.auto_disable":    "Auto-disable",
		"startup.webhook":         "Webhook",
		"startup.enabled":         "enabled",
		"startup.disabled":        "disabled",
		"startup.subnet_grouping":       "Subnet grouping",
		"startup.violation_threshold":   "Violation threshold",
		"startup.threshold_window":      "Threshold window",
		"startup.sec":                   "s",
	},
}

func SetLanguage(lang string) {
	if _, ok := translations[lang]; ok {
		current = lang
	}
}

func T(key string) string {
	if msg, ok := translations[current][key]; ok {
		return msg
	}
	if msg, ok := translations["en"][key]; ok {
		return msg
	}
	return key
}
