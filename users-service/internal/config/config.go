package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port string

	MongoURI string
	MongoDB  string

	JWTSecret string
	JWTTTL    time.Duration

	SMTPHost  string
	SMTPPort  string
	SMTPUser  string
	SMTPPass  string
	MailFrom  string
	ClientURL string

	DefaultQuotaBytes int64

	PasswordMaxAge time.Duration
	PasswordMinAge time.Duration

	ConfirmTTL time.Duration
	OTPTTL     time.Duration
	ResetTTL   time.Duration

	MaxFailedLogins int
	MaxOTPAttempts  int
	LockDuration    time.Duration

	AdminEmail    string
	AdminPassword string
}

type loader struct {
	errs []string
}

func (l *loader) str(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (l *loader) duration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		l.errs = append(l.errs, fmt.Sprintf("%s: neispravno trajanje %q", key, v))
		return def
	}
	return d
}

func (l *loader) integer(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		l.errs = append(l.errs, fmt.Sprintf("%s: neispravan broj %q", key, v))
		return def
	}
	return n
}

func (l *loader) integer64(key string, def int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		l.errs = append(l.errs, fmt.Sprintf("%s: neispravan broj %q", key, v))
		return def
	}
	return n
}

func Load() (*Config, error) {
	l := &loader{}
	c := &Config{
		Port:      l.str("PORT", "8080"),
		MongoURI:  l.str("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:   l.str("MONGO_DB", "users"),
		JWTSecret: os.Getenv("JWT_SECRET"),
		JWTTTL:    l.duration("JWT_TTL", 30*time.Minute),

		SMTPHost:  l.str("SMTP_HOST", "localhost"),
		SMTPPort:  l.str("SMTP_PORT", "1025"),
		SMTPUser:  os.Getenv("SMTP_USER"),
		SMTPPass:  os.Getenv("SMTP_PASS"),
		MailFrom:  l.str("MAIL_FROM", "noreply@drive.local"),
		ClientURL: strings.TrimRight(l.str("CLIENT_URL", "http://localhost:4200"), "/"),

		DefaultQuotaBytes: l.integer64("DEFAULT_QUOTA_BYTES", 1<<30),

		PasswordMaxAge: l.duration("PASSWORD_MAX_AGE", 60*24*time.Hour),
		PasswordMinAge: l.duration("PASSWORD_MIN_AGE", 24*time.Hour),

		ConfirmTTL: l.duration("CONFIRM_TTL", 24*time.Hour),
		OTPTTL:     l.duration("OTP_TTL", 5*time.Minute),
		ResetTTL:   l.duration("RESET_TTL", 15*time.Minute),

		MaxFailedLogins: l.integer("MAX_FAILED_LOGINS", 5),
		MaxOTPAttempts:  l.integer("MAX_OTP_ATTEMPTS", 5),
		LockDuration:    l.duration("LOCK_DURATION", 15*time.Minute),

		AdminEmail:    os.Getenv("ADMIN_EMAIL"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
	}

	if len(c.JWTSecret) < 32 {
		l.errs = append(l.errs, "JWT_SECRET mora imati najmanje 32 karaktera")
	}
	if len(l.errs) > 0 {
		return nil, errors.New(strings.Join(l.errs, "; "))
	}
	return c, nil
}
