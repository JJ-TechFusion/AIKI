package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	LinkedIn LinkedInConfig
	SerpAPI  SerpConfig
	AI       AIConfig
}

type AIConfig struct {
	OpenAI    OpenAIConfig
	Anthropic AnthropicConfig
}

type OpenAIConfig struct {
	APIKey       string
	DefaultModel string
}

type AnthropicConfig struct {
	APIKey       string
	DefaultModel string
}

type SerpConfig struct {
	Key string
}

type ServerConfig struct {
	Port string
	Env  string
}

type DatabaseConfig struct {
	URL      string
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	Addr     string
}

type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type LinkedInConfig struct {
	ClientID          string
	ClientSecret      string
	ClientCallbackUrl string
}

func Load() (*Config, error) {
	// Load .env file if it exists (ignore error in production)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Env:  getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			URL:      getEnv("DB_URL", ""),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "aiki"),
			Password: getEnv("DB_PASSWORD", "aiki_password"),
			DBName:   getEnv("DB_NAME", "aiki_db"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", ""),
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production"),
			AccessExpiry:  parseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"), 15*time.Minute),
			RefreshExpiry: parseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"), 7*24*time.Hour),
		},
		LinkedIn: LinkedInConfig{
			ClientID:          getEnv("LINKEDIN_CLIENT_ID", ""),
			ClientSecret:      getEnv("LINKEDIN_CLIENT_SECRET", ""),
			ClientCallbackUrl: getEnv("LINKEDIN_CALLBACK_URL", "http://localhost:%s/api/v1/auth/linkedin/callback"),
		},
		SerpAPI: SerpConfig{
			Key: getEnv("SERP_API_KEY", ""),
		},
		AI: AIConfig{
			OpenAI: OpenAIConfig{
				APIKey:       getEnv("OPENAI_API_KEY", ""),
				DefaultModel: getEnv("OPENAI_DEFAULT_MODEL", "gpt-4o-mini"),
			},
			Anthropic: AnthropicConfig{
				APIKey:       getEnv("ANTHROPIC_API_KEY", ""),
				DefaultModel: getEnv("ANTHROPIC_DEFAULT_MODEL", "claude-haiku-4-5-20251001"),
			},
		},
	}

	return cfg, nil
}

func (c *DatabaseConfig) ConnectionString() string {
	return c.URL

}

/*func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}*/

func (c *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(value string, defaultDuration time.Duration) time.Duration {
	d, err := time.ParseDuration(value)
	if err != nil {
		return defaultDuration
	}
	return d
}
