package services

import (
	"os"
	"strings"
)

func telegramAPIBase() string {
	base := strings.TrimRight(os.Getenv("TELEGRAM_API_BASE"), "/")
	if base == "" {
		return "https://api.telegram.org"
	}
	return base
}
