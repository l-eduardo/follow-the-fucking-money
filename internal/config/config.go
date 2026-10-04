package config

import (
	"os"
	"runtime"
	"strconv"
)

// Config holds runtime configuration loaded from environment or flags.
type Config struct {
	GraphURI     string
	GraphUser    string
	GraphPass    string
	DatabaseName string
	BatchSize    int
	MaxWorkers   int
	APIPort      int
	MetricsPort  int
	LogLevel     string
}

// LoadConfig loads configuration from environment variables with sensible defaults.
func LoadConfig() *Config {
	cfg := &Config{
		GraphURI:     getEnv("GRAPH_URI", "bolt://localhost:7687"),
		GraphUser:    getEnv("GRAPH_USER", ""),
		GraphPass:    getEnv("GRAPH_PASSWORD", ""),
		DatabaseName: getEnv("DATABASE_NAME", ""),
		BatchSize:    getEnvInt("BATCH_SIZE", 5000),
		MaxWorkers:   getEnvInt("MAX_WORKERS", runtime.NumCPU()*2),
		APIPort:      getEnvInt("API_PORT", 8075),
		MetricsPort:  getEnvInt("METRICS_PORT", 9090),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
	}
	return cfg
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
