// Package config loads gateway configuration from environment variables
// (and an optional .env file for local development) via Viper.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all environment-driven settings for the gateway service.
type Config struct {
	Environment string `mapstructure:"ENVIRONMENT"`
	LogLevel    string `mapstructure:"LOG_LEVEL"`

	HTTPPort string `mapstructure:"GATEWAY_HTTP_PORT"`

	InferenceGRPCHost string `mapstructure:"INFERENCE_GRPC_HOST"`
	InferenceGRPCPort string `mapstructure:"INFERENCE_GRPC_PORT"`
	InferenceTimeoutS int    `mapstructure:"INFERENCE_TIMEOUT_SECONDS"`

	// Postgres is not wired to any repository yet — there is no
	// persistence need until user accounts / medical logs land.
	// The DSN is parsed and held here so that addition is config-only.
	Postgres PostgresConfig `mapstructure:",squash"`

	// JWTSecret is unused by the current stub auth middleware. It is
	// read now so the env var contract is stable when real JWT/RBAC
	// auth is wired in.
	JWTSecret string `mapstructure:"JWT_SECRET"`
}

// PostgresConfig is a placeholder connection configuration. No
// repository currently uses it; it exists so the env contract and
// docker-compose wiring don't need to change when persistence is added.
type PostgresConfig struct {
	Host     string `mapstructure:"POSTGRES_HOST"`
	Port     string `mapstructure:"POSTGRES_PORT"`
	User     string `mapstructure:"POSTGRES_USER"`
	Password string `mapstructure:"POSTGRES_PASSWORD"`
	DBName   string `mapstructure:"POSTGRES_DB"`
	SSLMode  string `mapstructure:"POSTGRES_SSLMODE"`
}

// DSN builds a standard Postgres connection string from the placeholder
// config. Not called anywhere yet.
func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.DBName, p.SSLMode,
	)
}

// Load reads configuration from environment variables, with sane local
// defaults so the service is runnable without a .env file.
func Load() (*Config, error) {
	v := viper.New()

	v.SetDefault("ENVIRONMENT", "development")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("GATEWAY_HTTP_PORT", "8080")
	v.SetDefault("INFERENCE_GRPC_HOST", "localhost")
	v.SetDefault("INFERENCE_GRPC_PORT", "50051")
	v.SetDefault("INFERENCE_TIMEOUT_SECONDS", 30)
	v.SetDefault("POSTGRES_HOST", "localhost")
	v.SetDefault("POSTGRES_PORT", "5432")
	v.SetDefault("POSTGRES_USER", "omnimed")
	v.SetDefault("POSTGRES_PASSWORD", "omnimed")
	v.SetDefault("POSTGRES_DB", "omnimed")
	v.SetDefault("POSTGRES_SSLMODE", "disable")
	v.SetDefault("JWT_SECRET", "")

	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("../..")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("config: reading .env: %w", err)
		}
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshalling: %w", err)
	}

	return cfg, nil
}

// InferenceAddr returns the dial target for the AI inference gRPC server.
func (c *Config) InferenceAddr() string {
	return fmt.Sprintf("%s:%s", c.InferenceGRPCHost, c.InferenceGRPCPort)
}
