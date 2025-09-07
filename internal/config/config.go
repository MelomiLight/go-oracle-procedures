package config

import "os"

type Config struct {
	Server         *Server
	OracleDatabase *OracleDatabase
}

func NewConfig() *Config {
	return &Config{
		Server:         newServer(),
		OracleDatabase: newOracleDatabase(),
	}
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}
