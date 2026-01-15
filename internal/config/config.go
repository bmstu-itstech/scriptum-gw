package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Server struct {
	Addr      string `mapstructure:"addr"`
	JwtSecret string `mapstructure:"jwt_secret"`
}

type BoxesService struct {
	Addr string `mapstructure:"addr"`
}

type FileService struct {
	Addr string `mapstructure:"addr"`
}

type JobsService struct {
	Addr string `mapstructure:"addr"`
}

type Logging struct {
	Level string `mapstructure:"level"`
}

type Config struct {
	Server       Server       `mapstructure:"server"`
	BoxesService BoxesService `mapstructure:"boxes_service"`
	FileService  FileService  `mapstructure:"file_service"`
	JobsService  JobsService  `mapstructure:"jobs_service"`
	Logging      Logging      `mapstructure:"logging"`
}

func Load(path string) (*Config, error) {
	// Нетривиальный момент Viper, не описанный в документации, но описанный в
	// 	https://github.com/spf13/viper/issues/1797
	// Без viper.ExperimentalBindStruct() переменная окружения загружается только если она была указана в yaml конфиге.
	// Так например:
	// 	postgres:
	//    uri:
	// и SC_POSTGRES_URI работает корректно, а без пустого uri -- не читает переменную вовсе. Причём,
	//	viper.Get("postgres.uri")
	// работает исправно -- проблема именно в Unmarshall, который почему-то полагается на файл.
	// Решение -- viper.ExperimentalBindStruct().
	v := viper.NewWithOptions(viper.ExperimentalBindStruct())
	v.SetConfigFile(path)
	v.AutomaticEnv()
	v.SetEnvPrefix("SC")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config '%s': %w", path, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(err)
	}
	return cfg
}
