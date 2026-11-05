package config

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server     ServerConfig
	CORS       CORSConfig
	DB         DBConfig
	Session    SessionConfig
	Google     GoogleConfig
	Cloudinary CloudinaryConfig
	GoogleAI   GoogleAIConfig
}

type ServerConfig struct {
	Env  string
	Addr string
}

func (s ServerConfig) IsProduction() bool {
	return s.Env == "production"
}

type CORSConfig struct {
	AllowOrigins     string
	AllowMethods     string
	AllowHeaders     string
	AllowCredentials bool
	MaxAge           int
}

type DBConfig struct {
	MySQL MySQLConfig
	Mongo MongoConfig
}

type MySQLConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type MongoConfig struct {
	URI    string
	DBName string
}

type SessionConfig struct {
	Secret   string
	Secure   bool
	SameSite http.SameSite
}

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
}

type CloudinaryConfig struct {
	CloudName string
	APIKey    string
	APISecret string
}

// GoogleAIConfig configures the Google AI Studio (Gemini) client. The API key
// is deliberately not required at boot: without it the server still starts and
// the AI-backed features degrade instead of taking the whole app down.
type GoogleAIConfig struct {
	APIKey     string
	Model      string
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
}

// Enabled reports whether a Google AI Studio client can be built.
func (g GoogleAIConfig) Enabled() bool {
	return g.APIKey != ""
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	v := &validator{}

	env := getEnv("ENV", "development")
	isProd := env == "production"

	cfg := &Config{
		Server: ServerConfig{
			Env:  env,
			Addr: getEnv("ADDR", ":3000"),
		},
		CORS: CORSConfig{
			AllowOrigins:     v.require("ORIGIN"),
			AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
			AllowHeaders:     "Origin,Content-Type,Authorization",
			AllowCredentials: true,
			MaxAge:           300,
		},
		DB: DBConfig{
			MySQL: MySQLConfig{
				Host:     v.require("MYSQL_DB_HOST"),
				Port:     getEnv("MYSQL_DB_PORT", "3306"),
				User:     v.require("MYSQL_DB_USER"),
				Password: v.require("MYSQL_DB_PASSWORD"),
				Name:     v.require("MYSQL_DB_NAME"),
			},
			Mongo: MongoConfig{
				URI:    v.require("MONGO_DB_URI"),
				DBName: v.require("MONGO_DB_NAME"),
			},
		},
		Session: SessionConfig{
			Secret:   v.require("SESSION_SECRET"),
			Secure:   isProd,
			SameSite: http.SameSiteLaxMode,
		},
		Google: GoogleConfig{
			ClientID:     v.require("GOOGLE_CLIENT_ID"),
			ClientSecret: v.require("GOOGLE_CLIENT_SECRET"),
			CallbackURL:  v.require("GOOGLE_CALLBACK_URL"),
		},
		Cloudinary: CloudinaryConfig{
			CloudName: v.require("CLOUDINARY_CLOUD_NAME"),
			APIKey:    v.require("CLOUDINARY_API_KEY"),
			APISecret: v.require("CLOUDINARY_API_SECRET"),
		},
		GoogleAI: GoogleAIConfig{
			APIKey: os.Getenv("GEMINI_API_KEY"),
			// A lite tier is the right default here: condensing a form that
			// is already written needs no flagship reasoning, and the lite
			// models carry a larger free-tier allowance. gemini-2.5-* is
			// closed to new API keys, so the default must be a 3.x name.
			Model:   getEnv("GEMINI_MODEL", "gemini-3.5-flash-lite"),
			BaseURL: strings.TrimRight(getEnv("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta"), "/"),
			// Flash models routinely take tens of seconds under load, so a
			// ceiling much below this times out calls that would have
			// succeeded.
			Timeout:    time.Duration(getEnvInt("GEMINI_TIMEOUT_SECONDS", 60)) * time.Second,
			MaxRetries: getEnvInt("GEMINI_MAX_RETRIES", 2),
		},
	}

	if err := v.err(); err != nil {
		return nil, err
	}

	return cfg, nil
}

type validator struct {
	missing []string
}

func (v *validator) require(key string) string {
	val := os.Getenv(key)
	if val == "" {
		v.missing = append(v.missing, key)
	}
	return val
}

func (v *validator) err() error {
	if len(v.missing) == 0 {
		return nil
	}
	return fmt.Errorf("missing required environment variables: %v", v.missing)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return i
}
