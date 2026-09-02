package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Postgres     *Postgres
	Redis        *Redis
	Server       *Server
	Auth         *Auth
	Verification *Verification
	RBAC         *RBAC
	R2           *R2
}

type R2 struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicURL       string
}

type RBAC struct {
	SuperAdminAccount string
}

type Server struct {
	Port           uint16
	Host           string
	TrustedProxies []string
	AllowedOrigins []string
}

type Auth struct {
	AccessTokenSecret string
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
}

type Verification struct {
	Secret         string
	CodeTTL        time.Duration
	ResendInterval time.Duration
	IPWindow       time.Duration
	IPLimit        int
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

type SMTP struct {
	Host        string
	Port        uint16
	Username    string
	Password    string
	FromAddress string
	FromName    string
	TLSMode     string
	Timeout     time.Duration
}

type WorkerConfig struct {
	Redis              *Redis
	SMTP               *SMTP
	VerificationSecret string
	Concurrency        int
	R2PublicURL        string
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

	redisConfig, err := loadRedis()
	if err != nil {
		return nil, err
	}

	serverPort, err := parsePort("SERVER_PORT")
	if err != nil {
		return nil, err
	}
	serverHost, err := requiredEnv("SERVER_HOST")
	if err != nil {
		return nil, err
	}
	allowedOrigins, err := parseOrigins("CORS_ALLOWED_ORIGINS")
	if err != nil {
		return nil, err
	}

	accessTokenSecret, err := loadBase64Secret("AUTH_ACCESS_TOKEN_SECRET")
	if err != nil {
		return nil, err
	}
	accessTokenTTL, err := parseDuration("AUTH_ACCESS_TOKEN_TTL")
	if err != nil {
		return nil, err
	}
	refreshTokenTTL, err := parseDuration("AUTH_REFRESH_TOKEN_TTL")
	if err != nil {
		return nil, err
	}
	if accessTokenTTL >= refreshTokenTTL {
		return nil, errors.New("AUTH_ACCESS_TOKEN_TTL must be shorter than AUTH_REFRESH_TOKEN_TTL")
	}

	verificationSecret, err := loadVerificationSecret()
	if err != nil {
		return nil, err
	}
	verificationCodeTTL, err := parseDuration("VERIFICATION_CODE_TTL")
	if err != nil {
		return nil, err
	}
	verificationResendInterval, err := parseDuration("VERIFICATION_RESEND_INTERVAL")
	if err != nil {
		return nil, err
	}
	verificationIPWindow, err := parseDuration("VERIFICATION_IP_WINDOW")
	if err != nil {
		return nil, err
	}
	verificationIPLimit, err := parsePositiveInt("VERIFICATION_IP_LIMIT")
	if err != nil {
		return nil, err
	}
	if verificationResendInterval >= verificationCodeTTL {
		return nil, errors.New("VERIFICATION_RESEND_INTERVAL must be shorter than VERIFICATION_CODE_TTL")
	}

	r2Config, err := loadR2()
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
		Redis: redisConfig,
		Server: &Server{
			Port:           serverPort,
			Host:           serverHost,
			TrustedProxies: parseList(os.Getenv("TRUSTED_PROXIES")),
			AllowedOrigins: allowedOrigins,
		},
		Auth: &Auth{
			AccessTokenSecret: string(accessTokenSecret),
			AccessTokenTTL:    accessTokenTTL,
			RefreshTokenTTL:   refreshTokenTTL,
		},
		Verification: &Verification{
			Secret:         verificationSecret,
			CodeTTL:        verificationCodeTTL,
			ResendInterval: verificationResendInterval,
			IPWindow:       verificationIPWindow,
			IPLimit:        verificationIPLimit,
		},
		RBAC: &RBAC{
			SuperAdminAccount: strings.TrimSpace(os.Getenv("RBAC_SUPER_ADMIN_ACCOUNT")),
		},
		R2: r2Config,
	}, nil
}

func loadR2() (*R2, error) {
	accountID, err := requiredEnv("R2_ACCOUNT_ID")
	if err != nil {
		return nil, err
	}
	accessKeyID, err := requiredEnv("R2_ACCESS_KEY_ID")
	if err != nil {
		return nil, err
	}
	secretAccessKey, err := requiredEnv("R2_SECRET_ACCESS_KEY")
	if err != nil {
		return nil, err
	}
	bucket, err := requiredEnv("R2_BUCKET")
	if err != nil {
		return nil, err
	}
	publicURL, err := loadR2PublicURL()
	if err != nil {
		return nil, err
	}

	return &R2{
		AccountID:       accountID,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Bucket:          bucket,
		PublicURL:       publicURL,
	}, nil
}

func loadR2PublicURL() (string, error) {
	publicURL, err := requiredEnv("R2_PUBLIC_URL")
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(publicURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" || parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("R2_PUBLIC_URL must be an origin like https://img.example.com")
	}
	return publicURL, nil
}

func LoadWorker() (*WorkerConfig, error) {
	redisConfig, err := loadRedis()
	if err != nil {
		return nil, err
	}

	smtpHost, err := requiredEnv("SMTP_HOST")
	if err != nil {
		return nil, err
	}
	smtpPort, err := parsePort("SMTP_PORT")
	if err != nil {
		return nil, err
	}
	smtpFromAddress, err := requiredEnv("SMTP_FROM_ADDRESS")
	if err != nil {
		return nil, err
	}
	address, err := mail.ParseAddress(smtpFromAddress)
	if err != nil || !strings.EqualFold(address.Address, smtpFromAddress) {
		return nil, errors.New("SMTP_FROM_ADDRESS must be a valid email address without a display name")
	}

	smtpUsername := strings.TrimSpace(os.Getenv("SMTP_USERNAME"))
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	if (smtpUsername == "") != (smtpPassword == "") {
		return nil, errors.New("SMTP_USERNAME and SMTP_PASSWORD must be provided together")
	}

	smtpTLSMode := strings.ToLower(strings.TrimSpace(os.Getenv("SMTP_TLS_MODE")))
	switch smtpTLSMode {
	case "none", "starttls", "implicit":
	default:
		return nil, errors.New("SMTP_TLS_MODE must be one of none, starttls, or implicit")
	}

	smtpTimeout, err := parseDuration("SMTP_TIMEOUT")
	if err != nil {
		return nil, err
	}
	if smtpTimeout >= time.Minute {
		return nil, errors.New("SMTP_TIMEOUT must be shorter than 1m")
	}
	if smtpUsername != "" && smtpTLSMode == "none" {
		return nil, errors.New("SMTP_TLS_MODE cannot be none when SMTP authentication is enabled")
	}
	if smtpTLSMode == "none" && !isLoopbackHost(smtpHost) {
		return nil, errors.New("SMTP_TLS_MODE can be none only for a loopback SMTP server")
	}
	concurrency, err := parsePositiveInt("MAIL_QUEUE_CONCURRENCY")
	if err != nil {
		return nil, err
	}

	verificationSecret, err := loadVerificationSecret()
	if err != nil {
		return nil, err
	}

	r2PublicURL, err := loadR2PublicURL()
	if err != nil {
		return nil, err
	}

	return &WorkerConfig{
		Redis: redisConfig,
		SMTP: &SMTP{
			Host:        smtpHost,
			Port:        smtpPort,
			Username:    smtpUsername,
			Password:    smtpPassword,
			FromAddress: smtpFromAddress,
			FromName:    strings.TrimSpace(os.Getenv("SMTP_FROM_NAME")),
			TLSMode:     smtpTLSMode,
			Timeout:     smtpTimeout,
		},
		VerificationSecret: verificationSecret,
		Concurrency:        concurrency,
		R2PublicURL:        r2PublicURL,
	}, nil
}

func loadRedis() (*Redis, error) {
	host, err := requiredEnv("REDIS_HOST")
	if err != nil {
		return nil, err
	}
	port, err := parsePort("REDIS_PORT")
	if err != nil {
		return nil, err
	}
	database, err := strconv.Atoi(strings.TrimSpace(os.Getenv("REDIS_DATABASE")))
	if err != nil || database < 0 {
		return nil, errors.New("REDIS_DATABASE must be a non-negative integer")
	}
	tlsEnabled, err := strconv.ParseBool(strings.TrimSpace(os.Getenv("REDIS_TLS")))
	if err != nil {
		return nil, errors.New("REDIS_TLS must be a boolean")
	}

	return &Redis{
		Host:     host,
		Port:     port,
		Username: strings.TrimSpace(os.Getenv("REDIS_USER")),
		Password: os.Getenv("REDIS_PASSWORD"),
		Database: database,
		TLS:      tlsEnabled,
	}, nil
}

func loadVerificationSecret() (string, error) {
	decoded, err := loadBase64Secret("VERIFICATION_SECRET")
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func loadBase64Secret(name string) ([]byte, error) {
	value, err := requiredEnv(name)
	if err != nil {
		return nil, err
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(decoded) < 32 {
		return nil, fmt.Errorf("%s must be base64-encoded and contain at least 32 random bytes", name)
	}
	return decoded, nil
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

func parseDuration(name string) (time.Duration, error) {
	value, err := time.ParseDuration(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return value, nil
}

func parsePositiveInt(name string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}

func parseList(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}

func parseOrigins(name string) ([]string, error) {
	values := parseList(os.Getenv(name))
	if len(values) == 0 {
		return nil, fmt.Errorf("%s is required", name)
	}

	origins := make([]string, 0, len(values))
	for _, value := range values {
		parsed, err := url.Parse(value)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
			parsed.Host == "" || parsed.User != nil ||
			(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, fmt.Errorf("%s contains invalid origin %q", name, value)
		}
		origins = append(origins, parsed.Scheme+"://"+parsed.Host)
	}
	return origins, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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
