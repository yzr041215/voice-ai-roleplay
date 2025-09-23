package config

import "github.com/spf13/viper"

type Config struct {
	Port      string
	ServeName string
	MySQL     MySQLConfig
	Log       LogConfig
}
type LogConfig struct {
	Level int
}
type MySQLConfig struct {
	Dsn string
}

func NewConfig() *Config {
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.SetConfigName("config")
	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
	c := &Config{}
	if err := viper.Unmarshal(c); err != nil {
		panic(err)
	}
	return c
}
