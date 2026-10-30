// Package config loads bot settings from environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BotToken string
	LogLevel slog.Level
	AdminIDs []int64
}

func Load() (Config, error) {
	cfg := Config{BotToken: os.Getenv("BOT_TOKEN")}
	if cfg.BotToken == "" {
		return Config{}, errors.New("BOT_TOKEN is required")
	}

	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		if err := cfg.LogLevel.UnmarshalText([]byte(lvl)); err != nil {
			return Config{}, fmt.Errorf("LOG_LEVEL: %w", err)
		}
	}

	ids, err := parseIDs(os.Getenv("ADMIN_IDS"))
	if err != nil {
		return Config{}, fmt.Errorf("ADMIN_IDS: %w", err)
	}
	cfg.AdminIDs = ids

	return cfg, nil
}

func parseIDs(s string) ([]int64, error) {
	var ids []int64
	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
