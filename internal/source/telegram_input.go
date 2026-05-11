package source

import (
	"fmt"
	"net/url"
	"strings"
)

type TelegramChannelInput struct {
	Channel   string
	PublicURL string
}

func NormalizeTelegramChannelInput(input string) (TelegramChannelInput, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return TelegramChannelInput{}, fmt.Errorf("telegram channel is required")
	}
	if strings.HasPrefix(value, "@") {
		return telegramChannelFromName(strings.TrimPrefix(value, "@"))
	}
	if strings.Contains(value, "://") {
		return telegramChannelFromURL(value)
	}
	return telegramChannelFromName(value)
}

func telegramChannelFromName(name string) (TelegramChannelInput, error) {
	channel := strings.Trim(strings.TrimSpace(name), "/")
	if !validTelegramChannel(channel) {
		return TelegramChannelInput{}, fmt.Errorf("invalid telegram channel %q", name)
	}
	return TelegramChannelInput{Channel: channel, PublicURL: "https://t.me/s/" + channel}, nil
}

func telegramChannelFromURL(rawURL string) (TelegramChannelInput, error) {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		if err == nil {
			err = fmt.Errorf("url must include scheme and host")
		}
		return TelegramChannelInput{}, fmt.Errorf("invalid telegram channel url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return TelegramChannelInput{}, fmt.Errorf("invalid telegram channel url scheme %q", u.Scheme)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if isTelegramHost(u.Host) {
		if len(parts) == 1 {
			return telegramChannelFromName(parts[0])
		}
		if len(parts) == 2 && parts[0] == "s" {
			return telegramChannelFromName(parts[1])
		}
		return TelegramChannelInput{}, fmt.Errorf("invalid telegram channel url path %q", u.Path)
	}
	if len(parts) == 2 && parts[0] == "s" && validTelegramChannel(parts[1]) {
		u.RawQuery = ""
		u.Fragment = ""
		return TelegramChannelInput{Channel: parts[1], PublicURL: u.String()}, nil
	}
	return TelegramChannelInput{}, fmt.Errorf("invalid telegram channel host %q", u.Host)
}

func isTelegramHost(host string) bool {
	host = strings.ToLower(strings.TrimPrefix(host, "www."))
	return host == "t.me" || host == "telegram.me"
}

func validTelegramChannel(channel string) bool {
	if channel == "" {
		return false
	}
	for _, r := range channel {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			continue
		}
		return false
	}
	return true
}
