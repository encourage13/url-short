package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

const MaxCodeLength = 10

type Config struct {
	Port         string
	BaseURL      string
	AdminToken   string
	CodeLength   int
	DatabaseType string
	Postgres     PostgresConfig
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func Load() (Config, error) {
	cfg := Config{
		Port:         getEnv("SERVER_PORT", "8080"),
		AdminToken:   os.Getenv("ADMIN_TOKEN"),
		DatabaseType: getEnv("DATABASE_TYPE", "in-memory"),
		Postgres: PostgresConfig{
			Host:     getEnv("PG_HOST", "localhost"),
			Port:     getEnv("PG_PORT", "5432"),
			User:     os.Getenv("PG_USER"),
			Password: os.Getenv("PG_PASSWORD"),
			DBName:   os.Getenv("PG_DBNAME"),
			SSLMode:  getEnv("PG_SSLMODE", "disable"),
		},
	}
	cfg.BaseURL = getEnv("BASE_URL", "http://localhost:"+cfg.Port)

	codeLength, err := strconv.Atoi(getEnv("CODE_LENGTH", "6"))
	if err != nil || codeLength < 4 || codeLength > MaxCodeLength {
		return Config{}, fmt.Errorf("CODE_LENGTH должен быть числом от 4 до %d", MaxCodeLength)
	}
	cfg.CodeLength = codeLength

	switch cfg.DatabaseType {
	case "in-memory":
	case "postgres":
		if cfg.Postgres.User == "" || cfg.Postgres.DBName == "" {
			return Config{}, fmt.Errorf("для DATABASE_TYPE=postgres нужны PG_USER и PG_DBNAME")
		}
	default:
		return Config{}, fmt.Errorf("неизвестный DATABASE_TYPE %q (ожидается in-memory или postgres)", cfg.DatabaseType)
	}

	return cfg, nil
}

func (p PostgresConfig) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(p.User, p.Password),
		Host:     p.Host + ":" + p.Port,
		Path:     p.DBName,
		RawQuery: url.Values{"sslmode": {p.SSLMode}}.Encode(),
	}
	return u.String()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
