package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const defaultConfigPath = "configs/config.json"

type Config struct {
	Server ServerConfig `json:"server"`
	Log    LogConfig    `json:"log"`
}

type ServerConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"-"`
	WriteTimeout time.Duration `json:"-"`
	IdleTimeout  time.Duration `json:"-"`
}

type LogConfig struct {
	Level string `json:"level"`
}

type rawConfig struct {
	Server rawServerConfig `json:"server"`
	Log    LogConfig       `json:"log"`
}

type rawServerConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	ReadTimeout  string `json:"read_timeout"`
	WriteTimeout string `json:"write_timeout"`
	IdleTimeout  string `json:"idle_timeout"`
}

func Load() (Config, error) {
	path := envOrDefault("CONFIG_PATH", defaultConfigPath)

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read configuration file %q: %w", path, err)
	}

	var raw rawConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("decode configuration file %q: %w", path, err)
	}

	cfg, err := parse(raw)
	if err != nil {
		return Config{}, err
	}

	if err := applyEnvironmentOverrides(&cfg); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate configuration: %w", err)
	}

	return cfg, nil
}

func parse(raw rawConfig) (Config, error) {
	readTimeout, err := time.ParseDuration(raw.Server.ReadTimeout)
	if err != nil {
		return Config{}, fmt.Errorf("parse server read timeout: %w", err)
	}

	writeTimeout, err := time.ParseDuration(raw.Server.WriteTimeout)
	if err != nil {
		return Config{}, fmt.Errorf("parse server write timeout: %w", err)
	}

	idleTimeout, err := time.ParseDuration(raw.Server.IdleTimeout)
	if err != nil {
		return Config{}, fmt.Errorf("parse server idle timeout: %w", err)
	}

	return Config{
		Server: ServerConfig{
			Host:         raw.Server.Host,
			Port:         raw.Server.Port,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
		Log: raw.Log,
	}, nil
}

func applyEnvironmentOverrides(cfg *Config) error {
	if host, ok := os.LookupEnv("SERVER_HOST"); ok {
		cfg.Server.Host = host
	}

	if portValue, ok := os.LookupEnv("SERVER_PORT"); ok {
		port, err := strconv.Atoi(portValue)
		if err != nil {
			return fmt.Errorf("parse SERVER_PORT: %w", err)
		}

		cfg.Server.Port = port
	}

	if level, ok := os.LookupEnv("LOG_LEVEL"); ok {
		cfg.Log.Level = level
	}

	return nil
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
		return errors.New("log level must be debug, info, warn, or error")
	}

	return nil
}

func envOrDefault(name, fallback string) string {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}

	return value
}
