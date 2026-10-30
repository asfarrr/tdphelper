package config

import (
	"log/slog"
	"slices"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("BOT_TOKEN", "token")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("ADMIN_IDS", "1, 2,")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want debug", cfg.LogLevel)
	}
	if !slices.Equal(cfg.AdminIDs, []int64{1, 2}) {
		t.Errorf("AdminIDs = %v, want [1 2]", cfg.AdminIDs)
	}
}

func TestLoadDBPath(t *testing.T) {
	t.Setenv("BOT_TOKEN", "token")

	t.Setenv("DB_PATH", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBPath != "data/tdphelper.db" {
		t.Errorf("default DBPath = %q", cfg.DBPath)
	}

	t.Setenv("DB_PATH", "/var/lib/bot.db")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBPath != "/var/lib/bot.db" {
		t.Errorf("DBPath = %q", cfg.DBPath)
	}
}

func TestLoadRequiresToken(t *testing.T) {
	t.Setenv("BOT_TOKEN", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error without BOT_TOKEN")
	}
}
