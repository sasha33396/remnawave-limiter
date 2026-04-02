package telegram

import (
	"strings"
	"testing"
	"time"

	"github.com/remnawave/limiter/internal/api"
)

func TestFormatManualAlert(t *testing.T) {
	loc := time.UTC
	user := &api.CachedUser{
		UUID:            "uuid-123",
		UserID:          "1234",
		Username:        "testuser",
		Email:           "test@example.com",
		SubscriptionURL: "https://example.com/sub/123",
		HWIDDeviceLimit: 3,
	}
	ips := []api.ActiveIP{
		{IP: "1.1.1.1", NodeName: "Node-DE", NodeUUID: "n1"},
		{IP: "2.2.2.2", NodeName: "Node-US", NodeUUID: "n2"},
		{IP: "3.3.3.3", NodeName: "Node-NL", NodeUUID: "n3"},
		{IP: "4.4.4.4", NodeName: "Node-DE", NodeUUID: "n1"},
	}
	limit := 3

	result := FormatManualAlert(user, ips, limit, loc)

	checks := []struct {
		name    string
		want    string
	}{
		{"contains title", "Превышение лимита устройств"},
		{"contains username", "<code>testuser</code>"},
		{"contains email", "<code>test@example.com</code>"},
		{"contains limit", "3"},
		{"contains ip count", "4 IP"},
		{"contains ip1", "<code>1.1.1.1</code>"},
		{"contains ip2", "<code>2.2.2.2</code>"},
		{"contains ip3", "<code>3.3.3.3</code>"},
		{"contains ip4", "<code>4.4.4.4</code>"},
		{"contains node1", "Node-DE"},
		{"contains node2", "Node-US"},
		{"contains node3", "Node-NL"},
		{"contains subscription link", "https://example.com/sub/123"},
		{"contains profile link", "Профиль"},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(result, c.want) {
				t.Errorf("expected result to contain %q, got:\n%s", c.want, result)
			}
		})
	}
}

func TestFormatAutoAlert(t *testing.T) {
	loc := time.UTC
	user := &api.CachedUser{
		UUID:     "uuid-456",
		UserID:   "5678",
		Username: "autouser",
		Email:    "auto@example.com",
	}
	ips := []api.ActiveIP{
		{IP: "5.5.5.5", NodeName: "Node-FR", NodeUUID: "n5"},
		{IP: "6.6.6.6", NodeName: "Node-UK", NodeUUID: "n6"},
	}
	limit := 2
	duration := 30

	result := FormatAutoAlert(user, ips, limit, duration, loc)

	checks := []struct {
		name string
		want string
	}{
		{"contains auto title", "автоматически отключена"},
		{"contains username", "<code>autouser</code>"},
		{"contains email", "<code>auto@example.com</code>"},
		{"contains duration", "30 мин"},
		{"contains ip", "<code>5.5.5.5</code>"},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(result, c.want) {
				t.Errorf("expected result to contain %q, got:\n%s", c.want, result)
			}
		})
	}
}

func TestFormatAutoAlert_Permanent(t *testing.T) {
	loc := time.UTC
	user := &api.CachedUser{
		UUID:     "uuid-789",
		UserID:   "9012",
		Username: "permuser",
		Email:    "perm@example.com",
	}
	ips := []api.ActiveIP{
		{IP: "7.7.7.7", NodeName: "Node-JP", NodeUUID: "n7"},
	}

	result := FormatAutoAlert(user, ips, 1, 0, loc)

	if !strings.Contains(result, "Перманентно") {
		t.Errorf("expected result to contain 'Перманентно' for duration=0, got:\n%s", result)
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"<b>bold</b>", "&lt;b&gt;bold&lt;/b&gt;"},
		{"a & b", "a &amp; b"},
	}
	for _, tc := range tests {
		got := escapeHTML(tc.input)
		if got != tc.want {
			t.Errorf("escapeHTML(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFormatActionResult(t *testing.T) {
	tests := []struct {
		action   string
		admin    string
		username string
		want     string
	}{
		{"drop", "admin1", "user1", "✅ Подключения сброшены"},
		{"disable", "admin2", "user2", "🔒 Подписка отключена"},
		{"ignore", "admin3", "user3", "🔇 Добавлен в whitelist"},
		{"enable", "admin4", "user4", "🔓 Подписка включена"},
	}

	for _, tc := range tests {
		t.Run(tc.action, func(t *testing.T) {
			result := FormatActionResult(tc.action, tc.admin, tc.username)
			if !strings.Contains(result, tc.want) {
				t.Errorf("FormatActionResult(%q, %q, %q) = %q, want to contain %q",
					tc.action, tc.admin, tc.username, result, tc.want)
			}
			if !strings.Contains(result, tc.admin) {
				t.Errorf("expected result to contain admin name %q", tc.admin)
			}
		})
	}
}
