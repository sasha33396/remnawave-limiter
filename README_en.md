# Remnawave Limiter

**Centralized device control for Remnawave**

Automatic monitoring of simultaneous user connections from the Remnawave panel. Tracks IP addresses from all nodes via API, compares against each user's device limit, and notifies administrators via Telegram bot with instant management capabilities.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[Русская версия](README.md)

## Features

**Tracking:**
- Unique IP addresses over a configurable period
- Comparison with individual device limits from the panel (`hwidDeviceLimit`)
- Violations with tolerance support
- IP aggregation from all nodes — complete connection picture

**Admin notifications:**
- Limit and number of detected IPs
- Violation count over the last 24 hours
- List of all IP addresses with node names
- Link to user profile
- Delivery to chat, channel, group, or specific thread
- Inline buttons for instant actions

**Two operation modes:**
- **Manual** — alerts with buttons: drop connections, disable subscription, add to whitelist
- **Automatic** — auto-disable subscription for a set time or permanently, with auto-restore by timer

**Webhook notifications:**
- HTTP POST with full violation information (JSON)
- User data, IP addresses, limits, violation counter
- Optional authorization via secret header (`X-Webhook-Secret`)
- Works in both modes (manual and auto)

**Violation threshold:**
- Configurable threshold — action only after N violations within a given period
- Protection against false positives from brief limit spikes
- With `VIOLATION_THRESHOLD=1` — instant reaction (default behavior)

**Flexible settings:**
- Tolerance above the limit
- Cooldown between alerts
- Check interval and IP activity window
- User data caching
- Timezone selection
- Interface language (Russian / English)

## Architecture

```
┌─────────────┐     ┌──────────────────┐     ┌───────────────┐
│  Remnawave  │◄────│  remnawave-      │────►│   Telegram    │
│  Panel API  │     │  limiter         │     │   Bot API     │
└─────────────┘     │  (Go binary)     │     └───────────────┘
                    │  ┌────────────┐  │
                    │  │   Redis    │  │
                    │  └────────────┘  │
                    └──────────────────┘
```

- **Single instance** — install alongside the panel or on any server
- **No node installation required** — everything via the panel API
- **Docker Compose** — service + Redis in one file

### How it works

1. Fetches the list of active nodes via API (`GET /api/nodes`)
2. Requests user IPs from each node (in parallel)
3. Aggregates IPs per user across all nodes
4. Filters by last activity time (`lastSeen`)
5. Compares active IP count against the limit + tolerance
6. On violation — increments the threshold counter
7. If threshold is reached — reacts depending on the mode (alert or auto-block)

## Requirements

- Docker and Docker Compose
- Remnawave panel with an API token
- Telegram bot (create via [@BotFather](https://t.me/BotFather))

## Installation

### 1. Create directory

```bash
mkdir -p /opt/remnawave-limiter && cd /opt/remnawave-limiter
```

### 2. Download required files

```bash
curl -O https://raw.githubusercontent.com/syvlech/remnawave-limiter/master/docker-compose.yml
curl -O https://raw.githubusercontent.com/syvlech/remnawave-limiter/master/.env.example
```

### 3. Create configuration

```bash
cp .env.example .env
nano .env
```

Fill in the required parameters:

```bash
REMNAWAVE_API_URL=https://panel.example.com
REMNAWAVE_API_TOKEN=your-api-token-here
TELEGRAM_BOT_TOKEN=123456:ABC-DEF
TELEGRAM_CHAT_ID=-1001234567890
TELEGRAM_ADMIN_IDS=123456789
LANGUAGE=en
```

### 4. Start

```bash
docker compose pull
docker compose up -d
```

### Verify

```bash
docker compose logs -f limiter
```

### Update

```bash
cd /opt/remnawave-limiter
docker compose pull
docker compose up -d
```

## Configuration

All settings via `.env` file or environment variables.

| Parameter | Default | Description |
|-----------|:---:|-----------|
| `REMNAWAVE_API_URL` | **required** | Remnawave panel address |
| `REMNAWAVE_API_TOKEN` | **required** | API token (generated in the panel) |
| `TELEGRAM_BOT_TOKEN` | **required** | Bot token from @BotFather |
| `TELEGRAM_CHAT_ID` | **required** | Chat/channel/group ID for alerts |
| `TELEGRAM_ADMIN_IDS` | **required** | Admin IDs separated by commas (only they can press buttons) |
| `TELEGRAM_THREAD_ID` | — | Thread/topic ID in a supergroup |
| `CHECK_INTERVAL` | `30` | Check interval (seconds) |
| `ACTIVE_IP_WINDOW` | `300` | IP is considered active if `lastSeen` < this value (seconds) |
| `TOLERANCE` | `0` | Allowed excess over the limit. If limit is 3 and tolerance is 1, reaction at 5+ IPs |
| `COOLDOWN` | `300` | Cooldown between alerts for one user (seconds) |
| `USER_CACHE_TTL` | `600` | User data cache TTL (seconds) |
| `DEFAULT_DEVICE_LIMIT` | `0` | Default limit if user has no `hwidDeviceLimit`. 0 = no limit |
| `ACTION_MODE` | `manual` | `manual` — alert with buttons, `auto` — auto-disable subscription |
| `AUTO_DISABLE_DURATION` | `0` | Temporary disable duration in minutes. 0 = permanent only. In `manual` adds a button, in `auto` sets auto-restore time |
| `WEBHOOK_URL` | — | URL for sending webhooks on violations (POST JSON). Empty = disabled |
| `WEBHOOK_SECRET` | — | Secret for `X-Webhook-Secret` header (optional) |
| `WHITELIST_USER_IDS` | — | UUIDs to exclude from checks (comma-separated) |
| `VIOLATION_THRESHOLD` | `1` | Number of violations required before taking action. 1 = instant reaction |
| `VIOLATION_THRESHOLD_WINDOW` | `3600` | Time window in seconds for counting violations. Counter resets if no new violations occur within this period |
| `SUBNET_GROUPING` | `false` | Group IPs by /24 subnet (IPv4) and /48 (IPv6). When enabled, counts unique subnets instead of IPs — reduces false positives from CGNAT |
| `REDIS_URL` | `redis://redis:6379` | Redis address |
| `TIMEZONE` | `UTC` | Timezone for alert timestamps (e.g. `Europe/Moscow`) |
| `LANGUAGE` | `ru` | Interface language: `ru` or `en` |

## Limit logic

| `hwidDeviceLimit` | Behavior |
|:-:|-----------|
| `> 0` | Used as the device limit |
| `null` | Uses `DEFAULT_DEVICE_LIMIT` from config |
| `0` | No limit — user is skipped |

## Violation threshold

By default (`VIOLATION_THRESHOLD=1`) the limiter reacts to every detected violation. When the threshold is increased, the action (alert or auto-block) is only triggered after the required number of violations accumulates within the time window.

### How it works

1. Limit exceeded → cooldown is checked
2. Threshold counter is incremented (TTL = `VIOLATION_THRESHOLD_WINDOW`)
3. If counter < `VIOLATION_THRESHOLD` → logged, no action taken
4. If counter >= `VIOLATION_THRESHOLD` → action is triggered, counter resets

### Example

With `VIOLATION_THRESHOLD=3`, `VIOLATION_THRESHOLD_WINDOW=3600`, `COOLDOWN=300`:

| Time | Event | Counter | Action |
|:---:|-------|:---:|--------|
| 12:00 | Violation | 1/3 | Logged, no action |
| 12:05 | Violation | 2/3 | Logged, no action |
| 12:10 | Violation | 3/3 | Alert/block, counter reset |
| 12:15 | Violation | 1/3 | Logged, no action |

If more than `VIOLATION_THRESHOLD_WINDOW` seconds pass between violations, the counter resets automatically.

## Telegram bot

### Manual mode (`ACTION_MODE=manual`)

When the limit is exceeded, the bot sends an alert with buttons:

```
⚠️ Device limit exceeded

👤 User: username123
📊 Limit: 3 | Detected: 5 IP
📈 Violations in 24h: 3
🕐 2025-11-29 04:15:30

📍 IP addresses:
  • 10.0.1.10 (node: node-1)
  • 10.0.2.20 (node: node-1)
  • 10.0.3.30 (node: node-2)

🔗 Profile

[🔄 Drop connections] [🔒 Disable permanently]
[🔒 Disable for 10 min]        ← if AUTO_DISABLE_DURATION > 0
[🔇 Ignore]
```

| Button | Action |
|--------|--------|
| Drop connections | Reset active user connections via API |
| Disable permanently | Permanently deactivate subscription via API |
| Disable for N min | Temporarily deactivate with auto-restore by timer (shown when `AUTO_DISABLE_DURATION > 0`) |
| Ignore | Add to whitelist (no more alerts) |

### Automatic mode (`ACTION_MODE=auto`)

The subscription is disabled automatically, the bot sends an informational alert with an "Enable subscription" button.

If `AUTO_DISABLE_DURATION > 0` — the subscription is automatically restored by timer.

## Webhook

When a violation is detected, the limiter can send an HTTP POST request to the specified URL with full event information. The webhook works in both modes (manual and auto).

To enable, set `WEBHOOK_URL` in `.env`. Optionally, set `WEBHOOK_SECRET` — it will be sent in the `X-Webhook-Secret` header for verification on the receiving end.

**Example payload:**

```json
{
  "event": "violation_detected",
  "action_mode": "auto",
  "user": {
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": "123",
    "username": "john",
    "email": "john@example.com",
    "telegram_id": 123456789,
    "subscription_url": "https://panel.example.com/sub/abc"
  },
  "violation": {
    "ips": [
      {
        "ip": "1.2.3.4",
        "node_name": "DE-1",
        "node_uuid": "node-uuid-1",
        "last_seen": "2025-11-29T12:00:00Z"
      },
      {
        "ip": "5.6.7.8",
        "node_name": "US-1",
        "node_uuid": "node-uuid-2",
        "last_seen": "2025-11-29T12:01:00Z"
      }
    ],
    "ip_count": 5,
    "device_limit": 3,
    "tolerance": 1,
    "effective_limit": 4,
    "violation_count_24h": 3
  },
  "action": {
    "auto_disable_duration_min": 10
  },
  "timestamp": "2025-11-29T12:05:00Z"
}
```

| Field | Description |
|-------|-------------|
| `event` | Always `violation_detected` |
| `action_mode` | Operation mode: `manual` or `auto` |
| `user` | User data (UUID, username, email, telegram_id, subscription_url) |
| `violation.ips` | List of active IPs with node name and last activity time |
| `violation.ip_count` | Number of unique IPs |
| `violation.device_limit` | User's device limit |
| `violation.tolerance` | Tolerance value from config |
| `violation.effective_limit` | Effective limit (device_limit + tolerance) |
| `violation.violation_count_24h` | Number of violations in the last 24 hours |
| `action.auto_disable_duration_min` | Disable duration in minutes (0 = permanent) |
| `timestamp` | Violation detection time (ISO 8601) |

## FAQ

### How to find my Telegram Chat ID?

Add [@userinfobot](https://t.me/userinfobot) and send `/start`. For a group/channel — add the bot to the group and use the API or [@getidsbot](https://t.me/getidsbot).

### What happens if the panel API is unavailable?

The service logs an error, skips the current check cycle, and retries after `CHECK_INTERVAL` seconds. API requests are automatically retried up to 3 times with exponential backoff.

### Can I use Redis from Remnawave?

You can, but it's not recommended. The project runs its own Redis in Docker Compose. If you want to use an existing one — specify its address in `REDIS_URL`.

### How to add a user to the whitelist via the bot?

Press the "Ignore" button in an alert. The user will be added to the whitelist in Redis and will no longer be checked. The whitelist persists across restarts.

### How to change the limit for a specific user?

The limit is taken from the `hwidDeviceLimit` field in the Remnawave panel. Change it in the user's subscription settings.

## Support

- **Issues**: [GitHub Issues](https://github.com/syvlech/remnawave-limiter/issues)

## License

MIT License — see [LICENSE](LICENSE)
