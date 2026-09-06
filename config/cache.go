package config

import "time"

type Cache struct {
	Type           string        `mapstructure:"type"`
	RedisAddr      string        `mapstructure:"redis-addr"`
	RedisPwd       string        `mapstructure:"redis-pwd"`
	RedisDb        int           `mapstructure:"redis-db"`
	FileDir        string        `mapstructure:"file-dir"`
	ConnectTimeout time.Duration `mapstructure:"connect-timeout"`
}
