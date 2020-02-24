package pkg

import (
	"encoding/json"
	"fmt"
	logger "github.com/Sirupsen/logrus"
	"os"
	"strings"
)

const (
	ENVAccessKeyId     = "access_key_id"
	ENVAccessKeySecret = "access_key_secret"
	ENVLOGLEVEL        = "log_level"
	ENVDefaultSender   = "default_sender"
	ENVDefaultFilter   = "default_filter"
	ENVDefaultPattern  = "default_pattern"
	ENVNOTEMPTYFIELD   = "default_not_empty_field"
	ENVALARMCONFIG     = "alarm_config"

	DEFAULTSENDER      SenderName = "DINGDING"
	DEFAULTFILTERFIELD            = "price_per_core"
	DEFAULTCOLOR                  = true
)

// mandatory fields for each request
var (
	DEFAULTNOTEMPTYFIELD = []string{"sys.ding.conversationTitle", "region"}
)

type Config struct {
	AccessKeyId          string
	AccessKeySecret      string
	LogLevel             string
	DefaultSender        []*SenderConfig
	DefaultFilter        *FilterConfig
	DefaultPattern       *MessagePatternConfig
	DefaultNotEmptyField *[]string
}

type SenderConfig struct {
	SenderName SenderName         `json:"sender_name"`
	Token      *map[string]string `json:"token"`
}

type MessagePatternConfig struct {
	Color      *bool  `json:"color"`       // default true
	ColorField string `json:"color_field"` // default InstanceTypeId
}

type AlarmConfig struct {
	Cron          string                `json:"cron"`
	ConsStr       string                `json:"cons_str"`
	Cons          *Advisor              `json:"cons"`
	Filter        *FilterConfig         `json:"filter"`
	Sender        []*SenderConfig       `json:"sender"`
	Pattern       *MessagePatternConfig `json:"pattern"`
	NotEmptyField *[]string             `json:"not_empty_field"`
}

type FilterConfig struct {
	JudgedByField string `json:"judged_by_field"` // default price_per_core
	Threshold     string `json:"threshold"`
}

type EnvAlarmConfig *map[string][]AlarmConfig

var Cfg *Config

func init() {
	Cfg = &Config{}
	Cfg.AccessKeyId = os.Getenv(ENVAccessKeyId)
	Cfg.AccessKeySecret = os.Getenv(ENVAccessKeySecret)
	if len(Cfg.AccessKeyId) == 0 || len(Cfg.AccessKeySecret) == 0 {
		panic("config of AccessKeyId or config of AccessKeySecret EMPTY !")
	}

	Cfg.LogLevel = os.Getenv(ENVLOGLEVEL)
	ConfigLogger(&Cfg.LogLevel)

	token := make(map[string]string)
	defaultSender := &SenderConfig{
		SenderName: DEFAULTSENDER,
		Token:      &token, // default ding ding token
	}
	Cfg.DefaultSender = []*SenderConfig{defaultSender}
	if len(os.Getenv(ENVDefaultSender)) != 0 {
		err := json.Unmarshal([]byte(os.Getenv(ENVDefaultSender)), &Cfg.DefaultSender)
		if err != nil {
			fmt.Println(err.Error())
			panic(err)
		}
	}

	defaultColor := DEFAULTCOLOR
	defaultColorPtr := &defaultColor
	Cfg.DefaultPattern = &MessagePatternConfig{
		Color:      defaultColorPtr,
		ColorField: "InstanceTypeId",
	}
	if len(os.Getenv(ENVDefaultPattern)) != 0 {
		err := json.Unmarshal([]byte(os.Getenv(ENVDefaultPattern)), Cfg.DefaultPattern)
		if err != nil {
			fmt.Println(err.Error())
			panic(err)
		}
	}

	Cfg.DefaultFilter = &FilterConfig{
		JudgedByField: DEFAULTFILTERFIELD,
		Threshold:     "0.1",
	}
	if len(os.Getenv(ENVDefaultFilter)) != 0 {
		err := json.Unmarshal([]byte(os.Getenv(ENVDefaultFilter)), Cfg.DefaultFilter)
		if err != nil {
			fmt.Println(err.Error())
			panic(err)
		}
	}

	if Cfg.DefaultNotEmptyField == nil {
		Cfg.DefaultNotEmptyField = &[]string{}
	}
	*Cfg.DefaultNotEmptyField = DEFAULTNOTEMPTYFIELD
	if len(os.Getenv(ENVNOTEMPTYFIELD)) != 0 {
		*Cfg.DefaultNotEmptyField = strings.Split(os.Getenv(ENVNOTEMPTYFIELD), ",")
	}

	logger.WithFields(logger.Fields{"Cfg_LogLevel": Cfg.LogLevel, "Cfg_DefaultNotEmptyField": *Cfg.DefaultNotEmptyField,
		"Cfg_DefaultFilter": *Cfg.DefaultFilter, "Cfg_DefaultPattern": Cfg.DefaultPattern, "Cfg_Sender": Cfg.DefaultSender}).
		Info("init Cfg")
}

func newEnvAlarmConfig() (EnvAlarmConfig, error) {
	var envAlarm EnvAlarmConfig = &map[string][]AlarmConfig{}
	alarmJsonStr := os.Getenv(ENVALARMCONFIG)
	if len(alarmJsonStr) == 0 {
		return nil, fmt.Errorf("empty env: " + ENVALARMCONFIG)
	}

	err := json.Unmarshal([]byte(alarmJsonStr), envAlarm)
	if err != nil {
		logger.Errorf("json decode err:", err.Error())
		return nil, err
	}
	return envAlarm, err
}

func LoadAlarmConfig(inputAdvisor *Advisor) (EnvAlarmConfig, error) {
	envAlarm, err := newEnvAlarmConfig()
	if err != nil {
		return nil, err
	}
	logger.Infof("envAlarm is: %++v", envAlarm)
	for title, item := range *envAlarm {
		for k, v := range item {
			// if alarm config empty set default value from cfg
			if v.NotEmptyField == nil {
				v.NotEmptyField = &[]string{}
				*v.NotEmptyField = *Cfg.DefaultNotEmptyField
			}
			if v.Filter == nil {
				v.Filter = &FilterConfig{}
				*v.Filter = *Cfg.DefaultFilter
			}
			if v.Sender == nil {
				v.Sender = Cfg.DefaultSender
				//v.Sender = &SenderConfig{}
				//*v.Sender = *Cfg.DefaultSender
			}
			if v.Pattern == nil {
				v.Pattern = &MessagePatternConfig{}
				*v.Pattern = *Cfg.DefaultPattern
			}

			// merge input advisor params
			if inputAdvisor == nil {
				consBytes := []byte(v.ConsStr)
				v.Cons, err = NewAdvisor(&consBytes)
				if err != nil {
					logger.Errorf("new advisor", err.Error())
					continue
				}
			} else {
				newCons := &Advisor{}
				*newCons = *inputAdvisor
				err = json.Unmarshal([]byte(v.ConsStr), newCons)
				if err != nil {
					logger.Errorf("set advisor based on cons_str error", err.Error())
					continue
				}
				v.Cons = newCons
			}
			(*envAlarm)[title][k] = v
		}
	}

	logger.Infof("loaded envAlarm is: %++v", envAlarm)
	return envAlarm, nil
}
