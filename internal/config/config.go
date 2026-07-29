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
)

type Config struct {
	Server ServerConfig
	Log    LogConfig
	Auth   AuthConfig
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

type AuthConfig struct {
	JWTSecret string
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
		Auth: AuthConfig{
			JWTSecret: os.Getenv("JWT_SECRET"),
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

	if len(c.Auth.JWTSecret) < 32 {
		return errors.New(
			"JWT_SECRET must contain at least 32 characters",
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
