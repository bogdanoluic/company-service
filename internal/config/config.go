package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultServerHost         = "0.0.0.0"
	defaultServerPort         = 8080
	defaultServerReadTimeout  = 10 * time.Second
	defaultServerWriteTimeout = 10 * time.Second
	defaultServerIdleTimeout  = 60 * time.Second

	defaultLogLevel = "info"

	defaultDatabaseHost           = "localhost"
	defaultDatabasePort           = 5432
	defaultDatabaseUser           = "company_service"
	defaultDatabaseName           = "company_service"
	defaultDatabaseSSLMode        = "disable"
	defaultDatabaseConnectTimeout = 5 * time.Second
)

type Config struct {
	Server   ServerConfig
	Log      LogConfig
	Database DatabaseConfig
}

type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type LogConfig struct {
	Level string
}

type DatabaseConfig struct {
	Host           string
	Port           int
	User           string
	Password       string
	Name           string
	SSLMode        string
	ConnectTimeout time.Duration
}

func Load() (Config, error) {
	serverPort, err := intFromEnvironment(
		"SERVER_PORT",
		defaultServerPort,
	)
	if err != nil {
		return Config{}, err
	}

	readTimeout, err := durationFromEnvironment(
		"SERVER_READ_TIMEOUT",
		defaultServerReadTimeout,
	)
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := durationFromEnvironment(
		"SERVER_WRITE_TIMEOUT",
		defaultServerWriteTimeout,
	)
	if err != nil {
		return Config{}, err
	}

	idleTimeout, err := durationFromEnvironment(
		"SERVER_IDLE_TIMEOUT",
		defaultServerIdleTimeout,
	)
	if err != nil {
		return Config{}, err
	}

	databasePort, err := intFromEnvironment(
		"DATABASE_PORT",
		defaultDatabasePort,
	)
	if err != nil {
		return Config{}, err
	}

	connectTimeout, err := durationFromEnvironment(
		"DATABASE_CONNECT_TIMEOUT",
		defaultDatabaseConnectTimeout,
	)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Server: ServerConfig{
			Host: envOrDefault(
				"SERVER_HOST",
				defaultServerHost,
			),
			Port:         serverPort,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
		Log: LogConfig{
			Level: envOrDefault(
				"LOG_LEVEL",
				defaultLogLevel,
			),
		},
		Database: DatabaseConfig{
			Host: envOrDefault(
				"DATABASE_HOST",
				defaultDatabaseHost,
			),
			Port: databasePort,
			User: envOrDefault(
				"DATABASE_USER",
				defaultDatabaseUser,
			),
			Password: os.Getenv("DATABASE_PASSWORD"),
			Name: envOrDefault(
				"DATABASE_NAME",
				defaultDatabaseName,
			),
			SSLMode: envOrDefault(
				"DATABASE_SSL_MODE",
				defaultDatabaseSSLMode,
			),
			ConnectTimeout: connectTimeout,
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate configuration: %w", err)
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.Server.Host == "" {
		return errors.New("server host is required")
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return errors.New("server port must be between 1 and 65535")
	}

	if c.Server.ReadTimeout <= 0 {
		return errors.New("server read timeout must be positive")
	}

	if c.Server.WriteTimeout <= 0 {
		return errors.New("server write timeout must be positive")
	}

	if c.Server.IdleTimeout <= 0 {
		return errors.New("server idle timeout must be positive")
	}

	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		return errors.New(
			"log level must be debug, info, warn, or error",
		)
	}

	if c.Database.Host == "" {
		return errors.New("database host is required")
	}

	if c.Database.Port < 1 || c.Database.Port > 65535 {
		return errors.New(
			"database port must be between 1 and 65535",
		)
	}

	if c.Database.User == "" {
		return errors.New("database user is required")
	}

	if c.Database.Password == "" {
		return errors.New("database password is required")
	}

	if c.Database.Name == "" {
		return errors.New("database name is required")
	}

	switch c.Database.SSLMode {
	case "disable", "allow", "prefer",
		"require", "verify-ca", "verify-full":
	default:
		return errors.New("invalid database SSL mode")
	}

	if c.Database.ConnectTimeout <= 0 {
		return errors.New(
			"database connect timeout must be positive",
		)
	}

	return nil
}

func intFromEnvironment(name string, fallback int) (int, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}

	return parsed, nil
}

func durationFromEnvironment(
	name string,
	fallback time.Duration,
) (time.Duration, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}

	return parsed, nil
}

func envOrDefault(name, fallback string) string {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}

	return value
}
