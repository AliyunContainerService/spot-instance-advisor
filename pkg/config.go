package pkg

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
)

const (
	ENVDingDingTokens  = "DingDingTokens"
	ENVAccessKeyId     = "access_key_id"
	ENVAccessKeySecret = "access_key_secret"
	ENVLOGLEVEL        = "LogLevel"
	ENVALARMCONFIG     = "alarm_config"
)

type Config struct {
	DingDingTokensMap map[string]string
	AccessKeyId       string
	AccessKeySecret   string
	LogLevel          string
}

type AlarmConfig struct {
	ConsStr  string   `json:"cons_str"`
	Cons     *Advisor `json:"cons"`
	PriceMax float64  `json:"price_max"`
	Cron     string   `json:"cron"`
}

type EnvAlarmConfig *map[string][]AlarmConfig

var Cfg *Config

func init() {
	tokenJsonStr := os.Getenv(ENVDingDingTokens)
	Cfg = &Config{
		DingDingTokensMap: nil,
		AccessKeyId:       "",
		AccessKeySecret:   "",
		LogLevel:          "",
	}
	err := json.Unmarshal([]byte(tokenJsonStr), &Cfg.DingDingTokensMap)
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
	Cfg.AccessKeyId = os.Getenv(ENVAccessKeyId)
	Cfg.AccessKeySecret = os.Getenv(ENVAccessKeySecret)
	if len(Cfg.AccessKeyId) == 0 || len(Cfg.AccessKeySecret) == 0 {
		panic("cfg.AccessKeyId or cfg.AccessKeySecret EMPTY !")
	}
	Cfg.LogLevel = os.Getenv(ENVLOGLEVEL)
	ConfigLogger(Cfg.LogLevel)
}

func LoadAlarmConfig(initAdvisor *Advisor) (EnvAlarmConfig, error) {
	var envAlarm EnvAlarmConfig = &map[string][]AlarmConfig{}
	alarmJsonStr := os.Getenv(ENVALARMCONFIG)
	if len(alarmJsonStr) == 0 {
		fmt.Println("empty ", ENVALARMCONFIG)
		return nil, fmt.Errorf("empty ENV " + ENVALARMCONFIG)
	}

	fmt.Println("alarmJsonStr is:", alarmJsonStr)
	err := json.Unmarshal([]byte(alarmJsonStr), envAlarm)
	if err != nil {
		log.Errorf("json unmarshal err:", err.Error())
		return nil, err
	}

	// set default val
	for title, item := range *envAlarm {
		(*envAlarm)[title] = []AlarmConfig{}
		for _, v := range item {
			if initAdvisor == nil {
				v.Cons, err = NewAdvisor([]byte(v.ConsStr))
				if err != nil {
					log.Errorf("new advisor", err.Error())
					continue
				}
			} else {
				newCons := &Advisor{}
				*newCons = *initAdvisor
				err = json.Unmarshal([]byte(v.ConsStr), newCons)
				if err != nil {
					log.Errorf("new advisor", err.Error())
					continue
				}
				v.Cons = newCons
			}
			(*envAlarm)[title] = append((*envAlarm)[title], v)
		}
	}

	return envAlarm, nil
}
