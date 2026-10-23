package config

import (
	"net/url"
	"testing"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"SERVER_PORT", "BASE_URL", "ADMIN_TOKEN", "CODE_LENGTH", "DATABASE_TYPE",
		"PG_HOST", "PG_PORT", "PG_USER", "PG_PASSWORD", "PG_DBNAME", "PG_SSLMODE"} {
		t.Setenv(k, "")
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("SERVER_PORT", "9000")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "http://localhost:9000" || cfg.CodeLength != 6 || cfg.DatabaseType != "in-memory" {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}

func TestLoad_Invalid(t *testing.T) {
	tests := map[string]map[string]string{
		"code length too big":   {"CODE_LENGTH": "11"},
		"code length not int":   {"CODE_LENGTH": "six"},
		"unknown db type":       {"DATABASE_TYPE": "mysql"},
		"postgres without user": {"DATABASE_TYPE": "postgres", "PG_DBNAME": "db"},
	}
	for name, env := range tests {
		t.Run(name, func(t *testing.T) {
			clearEnv(t)
			for k, v := range env {
				t.Setenv(k, v)
			}
			if _, err := Load(); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestDSN_EscapesPassword(t *testing.T) {
	p := PostgresConfig{Host: "db", Port: "5432", User: "app", Password: "p@ss w/rd", DBName: "urls", SSLMode: "disable"}

	u, err := url.Parse(p.DSN())
	if err != nil {
		t.Fatal(err)
	}
	if pw, _ := u.User.Password(); pw != p.Password {
		t.Errorf("password = %q, want %q", pw, p.Password)
	}
}
