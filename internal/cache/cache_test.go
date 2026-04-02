package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/remnawave/limiter/internal/api"
)

const testRedisURL = "redis://localhost:6379/15"

func setupTestCache(t *testing.T) *Cache {
	t.Helper()

	c, err := New(testRedisURL)
	if err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}

	ctx := context.Background()
	if err := c.Ping(ctx); err != nil {
		c.Close()
		t.Skipf("Redis unavailable: %v", err)
	}

	// Flush test DB
	c.client.FlushDB(ctx)

	t.Cleanup(func() {
		c.client.FlushDB(ctx)
		c.Close()
	})

	return c
}

func TestCache_UserData(t *testing.T) {
	c := setupTestCache(t)
	ctx := context.Background()

	user := &api.CachedUser{
		UUID:            "uuid-123",
		UserID:          "user-456",
		Username:        "testuser",
		Email:           "test@example.com",
		TelegramID:      789,
		HWIDDeviceLimit: 3,
		Status:          "active",
		SubscriptionURL: "https://example.com/sub",
	}

	// Get non-existent user returns nil, nil
	got, err := c.GetUser(ctx, "user-456")
	if err != nil {
		t.Fatalf("GetUser error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for non-existent user, got %+v", got)
	}

	// Set user
	if err := c.SetUser(ctx, "user-456", user, 10*time.Second); err != nil {
		t.Fatalf("SetUser error: %v", err)
	}

	// Get user and verify fields
	got, err = c.GetUser(ctx, "user-456")
	if err != nil {
		t.Fatalf("GetUser error: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.UUID != user.UUID {
		t.Errorf("UUID = %q, want %q", got.UUID, user.UUID)
	}
	if got.UserID != user.UserID {
		t.Errorf("UserID = %q, want %q", got.UserID, user.UserID)
	}
	if got.Username != user.Username {
		t.Errorf("Username = %q, want %q", got.Username, user.Username)
	}
	if got.Email != user.Email {
		t.Errorf("Email = %q, want %q", got.Email, user.Email)
	}
	if got.TelegramID != user.TelegramID {
		t.Errorf("TelegramID = %d, want %d", got.TelegramID, user.TelegramID)
	}
	if got.HWIDDeviceLimit != user.HWIDDeviceLimit {
		t.Errorf("HWIDDeviceLimit = %d, want %d", got.HWIDDeviceLimit, user.HWIDDeviceLimit)
	}
	if got.Status != user.Status {
		t.Errorf("Status = %q, want %q", got.Status, user.Status)
	}
	if got.SubscriptionURL != user.SubscriptionURL {
		t.Errorf("SubscriptionURL = %q, want %q", got.SubscriptionURL, user.SubscriptionURL)
	}
}

func TestCache_Cooldown(t *testing.T) {
	c := setupTestCache(t)
	ctx := context.Background()

	// No cooldown initially
	active, err := c.IsCooldownActive(ctx, "user-1")
	if err != nil {
		t.Fatalf("IsCooldownActive error: %v", err)
	}
	if active {
		t.Fatal("expected no cooldown initially")
	}

	// Set cooldown
	if err := c.SetCooldown(ctx, "user-1", 10*time.Second); err != nil {
		t.Fatalf("SetCooldown error: %v", err)
	}

	// Check active
	active, err = c.IsCooldownActive(ctx, "user-1")
	if err != nil {
		t.Fatalf("IsCooldownActive error: %v", err)
	}
	if !active {
		t.Fatal("expected cooldown to be active")
	}
}

func TestCache_Whitelist(t *testing.T) {
	c := setupTestCache(t)
	ctx := context.Background()

	// Not whitelisted initially
	ok, err := c.IsWhitelisted(ctx, "user-1")
	if err != nil {
		t.Fatalf("IsWhitelisted error: %v", err)
	}
	if ok {
		t.Fatal("expected not whitelisted initially")
	}

	// Add to whitelist
	if err := c.AddToWhitelist(ctx, "user-1"); err != nil {
		t.Fatalf("AddToWhitelist error: %v", err)
	}

	// Check whitelisted
	ok, err = c.IsWhitelisted(ctx, "user-1")
	if err != nil {
		t.Fatalf("IsWhitelisted error: %v", err)
	}
	if !ok {
		t.Fatal("expected whitelisted after add")
	}

	// Remove from whitelist
	if err := c.RemoveFromWhitelist(ctx, "user-1"); err != nil {
		t.Fatalf("RemoveFromWhitelist error: %v", err)
	}

	// Check not whitelisted after removal
	ok, err = c.IsWhitelisted(ctx, "user-1")
	if err != nil {
		t.Fatalf("IsWhitelisted error: %v", err)
	}
	if ok {
		t.Fatal("expected not whitelisted after removal")
	}

	// InitWhitelist
	if err := c.InitWhitelist(ctx, []string{"a", "b", "c"}); err != nil {
		t.Fatalf("InitWhitelist error: %v", err)
	}
	for _, id := range []string{"a", "b", "c"} {
		ok, err = c.IsWhitelisted(ctx, id)
		if err != nil {
			t.Fatalf("IsWhitelisted(%q) error: %v", id, err)
		}
		if !ok {
			t.Fatalf("expected %q to be whitelisted after InitWhitelist", id)
		}
	}
}

func TestCache_RestoreTimer(t *testing.T) {
	c := setupTestCache(t)
	ctx := context.Background()

	// Set timer that expires in 1 second
	if err := c.SetRestoreTimer(ctx, "uuid-abc", 1*time.Second); err != nil {
		t.Fatalf("SetRestoreTimer error: %v", err)
	}

	// No expired timers yet
	expired, err := c.GetExpiredRestoreTimers(ctx)
	if err != nil {
		t.Fatalf("GetExpiredRestoreTimers error: %v", err)
	}
	if len(expired) != 0 {
		t.Fatalf("expected no expired timers, got %v", expired)
	}

	// Wait for expiry
	time.Sleep(1500 * time.Millisecond)

	// Now should be expired
	expired, err = c.GetExpiredRestoreTimers(ctx)
	if err != nil {
		t.Fatalf("GetExpiredRestoreTimers error: %v", err)
	}
	if len(expired) != 1 || expired[0] != "uuid-abc" {
		t.Fatalf("expected [uuid-abc], got %v", expired)
	}

	// Should be removed after retrieval
	expired, err = c.GetExpiredRestoreTimers(ctx)
	if err != nil {
		t.Fatalf("GetExpiredRestoreTimers error: %v", err)
	}
	if len(expired) != 0 {
		t.Fatalf("expected empty after retrieval, got %v", expired)
	}
}

// Verify that New returns an error for an invalid URL (not a skip - this doesn't need Redis)
func TestCache_InvalidURL(t *testing.T) {
	_, err := New("not-a-valid-url://bad")
	if err == nil {
		t.Fatal("expected error for invalid Redis URL")
	}
}

// Verify the internal redis client type is correct
func TestCache_ClientType(t *testing.T) {
	c := setupTestCache(t)
	_ = c.client // just ensure it's accessible as *redis.Client
	var _ *redis.Client = c.client
}
