package telegram

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

func buildProxyDialer(proxyURL string) (fasthttp.DialFunc, string, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, "", fmt.Errorf("не удалось разобрать URL прокси: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme == "" || u.Host == "" {
		return nil, "", fmt.Errorf("URL прокси должен содержать схему и host:port (например socks5://user:pass@host:1080)")
	}

	authHost := u.Host
	if u.User != nil {
		authHost = u.User.String() + "@" + u.Host
	}

	safeURL := scheme + "://"
	if u.User != nil {
		if _, hasPwd := u.User.Password(); hasPwd {
			safeURL += u.User.Username() + ":***@"
		} else {
			safeURL += u.User.Username() + "@"
		}
	}
	safeURL += u.Host

	switch scheme {
	case "http", "https":
		return fasthttpproxy.FasthttpHTTPDialerTimeout(authHost, 30*time.Second), safeURL, nil
	case "socks5", "socks5h":
		return fasthttpproxy.FasthttpSocksDialer(scheme + "://" + authHost), safeURL, nil
	default:
		return nil, "", fmt.Errorf("неподдерживаемая схема прокси %q (используйте http, https или socks5)", u.Scheme)
	}
}
