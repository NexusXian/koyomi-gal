package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Postgres *Postgres
	Redis    *Redis
    Server   *Server
}

type Server struct {
    Port uint16
    Host string
}

type Postgres struct {
	Host     string
	Port     uint16
	User     string
	Password string
	Database string
	SSLMode  string
}

type Redis struct {
	Host     string
	Port     uint16
	Username string
	Password string
	Database int
	TLS      bool
}

func Load() (*Config, error) {
	postgresHost, err := requiredEnv("POSTGRES_HOST")
	if err != nil {
		return nil, err
	}
	postgresPort, err := parsePort("POSTGRES_PORT")
	if err != nil {
		return nil, err
	}
	postgresUser, err := requiredEnv("POSTGRES_USER")
	if err != nil {
		return nil, err
	}
	postgresPassword, err := requiredEnv("POSTGRES_PASSWORD")
	if err != nil {
		return nil, err
	}
	postgresDatabase, err := requiredEnv("POSTGRES_DATABASE")
	if err != nil {
		return nil, err
	}
	postgresSSLMode, err := requiredEnv("POSTGRES_SSL_MODE")
	if err != nil {
		return nil, err
	}
	if !validPostgresSSLMode(postgresSSLMode) {
		return nil, fmt.Errorf("POSTGRES_SSL_MODE has invalid value %q", postgresSSLMode)
	}

	redisHost, err := requiredEnv("REDIS_HOST")
	if err != nil {
		return nil, err
	}
	redisPort, err := parsePort("REDIS_PORT")
	if err != nil {
		return nil, err
	}
	redisDatabase, err := strconv.Atoi(strings.TrimSpace(os.Getenv("REDIS_DATABASE")))
	if err != nil || redisDatabase < 0 {
		return nil, errors.New("REDIS_DATABASE must be a non-negative integer")
	}
	redisTLS, err := strconv.ParseBool(strings.TrimSpace(os.Getenv("REDIS_TLS")))
	if err != nil {
		return nil, errors.New("REDIS_TLS must be a boolean")
	}

    serverPort, err := parsePort("SERVER_PORT")
    if err != nil {
        return nil, err
    }
    serverHost, err := requiredEnv("SERVER_HOST")
    if err != nil {
        return nil, err
    }

	return &Config{
		Postgres: &Postgres{
			Host:     postgresHost,
			Port:     postgresPort,
			User:     postgresUser,
			Password: postgresPassword,
			Database: postgresDatabase,
			SSLMode:  postgresSSLMode,
		},
		Redis: &Redis{
			Host:     redisHost,
			Port:     redisPort,
			Username: os.Getenv("REDIS_USER"),
			Password: os.Getenv("REDIS_PASSWORD"),
			Database: redisDatabase,
			TLS:      redisTLS,
		},
        Server: &Server{
            Port: serverPort,
            Host: serverHost,
        },
	}, nil
}

func requiredEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}

func parsePort(name string) (uint16, error) {
	value, err := strconv.ParseUint(strings.TrimSpace(os.Getenv(name)), 10, 16)
	if err != nil || value == 0 {
		return 0, fmt.Errorf("%s must be an integer between 1 and 65535", name)
	}
	return uint16(value), nil
}

func validPostgresSSLMode(value string) bool {
	switch value {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
		return true
	default:
		return false
	}
}

func LoadEnv() error {
	environment := strings.TrimSpace(os.Getenv("APP_ENV"))
	if environment == "" {
		environment = "development"
	}

	err := godotenv.Load(".env." + environment)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
