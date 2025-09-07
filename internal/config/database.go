package config

import (
	"strconv"

	goora "github.com/sijms/go-ora/v2"
)

type OracleDatabase struct {
	Host     string
	Port     string
	User     string
	Password string
	Sid      string
}

func newOracleDatabase() *OracleDatabase {
	return &OracleDatabase{
		Host:     getEnv("ORACLE_HOST", "localhost"),
		Port:     getEnv("ORACLE_PORT", "1521"),
		User:     getEnv("ORACLE_USER", "app"),
		Password: getEnv("ORACLE_PASSWORD", "password"),
		Sid:      getEnv("ORACLE_SID", "FREEPDB1"),
	}
}

func (o *OracleDatabase) DSN() string {
	portInt, err := strconv.Atoi(o.Port)
	if err != nil {
		panic(err)
	}

	return goora.BuildUrl(o.Host, portInt, o.Sid, o.User, o.Password, nil)
}
