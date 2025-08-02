package config

import (
	"log"
	"os"
	"time"

	"github.com/goccy/go-yaml"
)

type config struct {
	Port            string
	ContextPath     string           `yaml:"context_path"`
	CacheDir        string           `yaml:"cache_dir"`
	CorsOrigin      string           `yaml:"cors_origin"`
	Users           []User           `yaml:"users"`
	SubscribeGroups []SubscribeGroup `yaml:"subscribe_groups"`
}

type User struct {
	Name            string               `yaml:"name"`
	Token           string               `yaml:"token"`
	SubscribeGroups []UserSubscribeGroup `yaml:"subscribe_groups"`
}

type UserSubscribeGroup struct {
	Name       string   `yaml:"name"`
	Subscribes []string `yaml:"subscribes"`
}

type SubscribeGroup struct {
	Name                string          `yaml:"name"`
	RequestHeaders      []RequestHeader `yaml:"request_headers"`
	PassResponseHeaders []string        `yaml:"pass_response_headers"`
	Subscribes          []Subscribe     `yaml:"subscribes"`
}

type Subscribe struct {
	Name string `yaml:"name"`
	Url  string `yaml:"url"`
}

type RequestHeader struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type CachedResponse struct {
	ExpireAt time.Time           `json:"expire_at"`
	Headers  map[string][]string `json:"headers"`
	Body     []byte              `json:"body"`
}

var Config = &config{}

func InitConfig() error {
	configBytes, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Read config failed, %v\n", err)
		return err
	}

	err = yaml.Unmarshal(configBytes, &Config)
	if err != nil {
		log.Fatalf("Decode config failed: %v\n", err)
	}

	return nil
}
