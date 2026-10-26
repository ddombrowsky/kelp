package postgresdb

import (
	"os"
	"strconv"
)

// MakeTestConfig returns the config tests use to reach postgres, taken from the POSTGRES_HOST, POSTGRES_PORT,
// POSTGRES_USER and POSTGRES_PASSWORD environment variables, defaulting to localhost:5432
func MakeTestConfig(dbName string) *Config {
	port := uint16(5432)
	if p, e := strconv.ParseUint(os.Getenv("POSTGRES_PORT"), 10, 16); e == nil {
		port = uint16(p)
	}

	return &Config{
		Host:      os.Getenv("POSTGRES_HOST"),
		Port:      port,
		DbName:    dbName,
		User:      os.Getenv("POSTGRES_USER"),
		Password:  os.Getenv("POSTGRES_PASSWORD"),
		SSLEnable: false,
	}
}
