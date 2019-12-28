package pkg

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	ENVDingDingTokens  = "DingDingTokens"
	ENVAccessKeyId     = "access_key_id"
	ENVAccessKeySecret = "access_key_secret"
	ENVLOGLEVEL        = "LogLevel"
)

type Config struct {
	DingDingTokensMap map[string]string
	AccessKeyId       string
	AccessKeySecret   string
	LogLevel          string
}

func LoadConfig() *Config {
	var cfg Config
	tokenJsonStr := os.Getenv(ENVDingDingTokens)
	err := json.Unmarshal([]byte(tokenJsonStr), &cfg.DingDingTokensMap)
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
	cfg.AccessKeyId = os.Getenv(ENVAccessKeyId)
	cfg.AccessKeySecret = os.Getenv(ENVAccessKeySecret)
	if len(cfg.AccessKeyId) == 0 || len(cfg.AccessKeySecret) == 0 {
		panic("cfg.AccessKeyId or cfg.AccessKeySecret EMPTY !")
	}
	cfg.LogLevel = os.Getenv(ENVLOGLEVEL)
	return &cfg
}
