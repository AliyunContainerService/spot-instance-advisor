package handler

import (
	"fmt"
	"github.com/IrisIris/spot-instance-advisor/pkg"
	logger "github.com/Sirupsen/logrus"
	"github.com/robfig/cron"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var ALARMNOTEMPTYFIELD = []string{"sys.ding.conversationTitle", "region"}

func judgeAlarmsByPolling(coversationAlarm *map[string][]pkg.AlarmConfig) {
	for convTitle, envConf := range *coversationAlarm {
		dingDingTokens, err := GetDingDingTokensByConvTitle(convTitle, pkg.Cfg)
		if dingDingTokens == nil || err != nil {
			logger.WithFields(logger.Fields{
				"convTitle": convTitle,
				"pkg.Cfg":   pkg.Cfg,
			}).Errorf("get dingdingTokens by conv title error: " + err.Error())
			continue
		}
		// new advisor
		for _, conf := range envConf {
			if conf.Cons == nil {
				continue
			}
			if len(conf.Cron) == 0 {
				continue
			}
			cron, cons, priceMax := conf.Cron, conf.Cons, conf.PriceMax
			fmt.Printf("cron job start ,cron: %s, cons:%s, dingDingTokens:%s, priceMax:%s", cron, cons, dingDingTokens, priceMax)
			//go judgeOneByPolling(cons, dingDingTokens, priceMax)
			go func(cons *pkg.Advisor, dingDingTokens *[]string, priceMax float64) {
				nowRes := &[]pkg.AdvisorResponse{}
				showed := &[]pkg.AdvisorChangedInstance{}
				newChanged := &[]pkg.AdvisorResponse{}
				var startTime *int64
				startTime = new(int64)
				*startTime = time.Now().Unix()
				for {
					judgeOne(cons, dingDingTokens, priceMax, nowRes, showed, newChanged, startTime)
					time.Sleep(5 * time.Second)
				}
			}(cons, dingDingTokens, priceMax)
		}
	}
}

func judgeOneByPolling(cons *pkg.Advisor, dingDingTokens *[]string, priceMax float64) {
	nowRes := &[]pkg.AdvisorResponse{}
	showed := &[]pkg.AdvisorChangedInstance{}
	newChanged := &[]pkg.AdvisorResponse{}
	var startTime *int64
	startTime = new(int64)
	*startTime = time.Now().Unix()
	for {
		judgeOne(cons, dingDingTokens, priceMax, nowRes, showed, newChanged, startTime)
		time.Sleep(5 * time.Second)
	}
}

func AlarmHandler(w http.ResponseWriter, r *http.Request) {
	vals := url.Values{}
	if r != nil {
		vals = r.URL.Query()
	}

	resp := &DingDingRep{
		Success:   true,
		ErrorCode: strconv.Itoa(int(ErrorNone)),
		ErrorMsg:  "",
		Fields:    nil,
	}

	// check && get json byte
	jsonBytes, err := InputCommonCheck(w, vals, pkg.Cfg, nil)
	if jsonBytes == nil || err != nil {
		logger.WithFields(logger.Fields{
			"vals":          vals,
			"notEmptyField": nil,
			"error":         err,
		}).Errorf("input common check error :")
		resp := PackageResp(false, "InputCommonCheck: "+err.Error(), nil)
		sendResponse(w, resp)
		return
	}
	// get new advisor
	initAdvisor, err := pkg.NewAdvisor(jsonBytes)
	if err != nil {
		logger.WithFields(logger.Fields{
			"advisor": *initAdvisor,
			"error":   err,
		}).Errorf("new advisor error :")
		resp := PackageResp(false, "new advisor err: "+err.Error(), nil)
		sendResponse(w, resp)
		return
	}
	fmt.Println("initAdvisor is", initAdvisor)

	// get alarm config from env
	envAlarmConfig, err := pkg.LoadAlarmConfig(initAdvisor)
	if err != nil {
		logger.Errorf("load alarm config error: " + err.Error())
		resp := PackageResp(false, "load alarm config error: "+err.Error(), nil)
		sendResponse(w, resp)
		return
	}

	// if sys.ding.conversationTitle is specificed
	coversationAlarm := envAlarmConfig
	if len(vals["sys.ding.conversationTitle"]) != 0 {
		coversationAlarm, err = GetAlarmConfigByConv(vals["sys.ding.conversationTitle"], envAlarmConfig)
		if err != nil {
			logger.Errorf("get alarm config by conv error: " + err.Error())
			return
		}
	}

	judgeAlarmsByCron(coversationAlarm)
	//judgeAlarmsByPolling(coversationAlarm)
	resp.Fields = map[string]string{
		"title": ALARMTITLE,
	}
	sendResponse(w, resp)
}

func GetAlarmConfigByConv(converTitiles []string, cfg pkg.EnvAlarmConfig) (pkg.EnvAlarmConfig, error) {
	var alarmConfig pkg.EnvAlarmConfig = &map[string][]pkg.AlarmConfig{}
	for _, name := range converTitiles {
		alarmConf, ok := (*cfg)[name]
		if !ok || len(alarmConf) == 0 {
			logger.Errorf("%s's EnvAlarmConfig  EMPTY!", name)
			return nil, fmt.Errorf("%s's EnvAlarmConfig  EMPTY!", name)
		} else {
			(*alarmConfig)[name] = alarmConf
		}
	}
	return alarmConfig, nil
}

func getAlarmsByReq(advisor *pkg.Advisor, priceMax float64) (*[]pkg.AdvisorResponse, Querytype) {
	qType := Alarm

	// get current value
	spotInstancePrices, err := advisor.SpotPricesAnalysis()
	if err != nil {
		logger.WithFields(logger.Fields{
			"spotInstancePrices": spotInstancePrices,
			"error":              err,
		}).Errorf("get spotInstancePrices error :")
		return nil, qType
	}

	if len(spotInstancePrices) == 0 {
		logger.WithFields(logger.Fields{
			"spotInstancePrices": spotInstancePrices,
			"error":              nil,
		}).Errorf("get empty spotInstancePrices :")
		return nil, qType
	}
	sort.Sort(spotInstancePrices)

	toShow := []pkg.AdvisorResponse{}
	count := 0
	for _, item := range spotInstancePrices {
		if count >= advisor.Limit {
			break
		}
		toShow = append(toShow, pkg.AdvisorResponse{
			InstanceTypeId: strings.Replace(item.InstanceTypeId, "ecs.", "", 1),
			ZoneId:         strings.Replace(item.ZoneId, advisor.Region+"-", "", -1),
			PricePerCore:   pkg.Decimal(item.PricePerCore),
		})
		count++
	}

	if len(toShow) == 0 {
		qType = Recover
		count = 0
		for _, item := range spotInstancePrices {
			if count >= advisor.Limit {
				break
			}
			toShow = append(toShow, pkg.AdvisorResponse{
				InstanceTypeId: strings.Replace(item.InstanceTypeId, "ecs.", "", 1),
				ZoneId:         strings.Replace(item.ZoneId, advisor.Region+"-", "", -1),
				PricePerCore:   pkg.Decimal(item.PricePerCore),
			})
			count++
		}
	}
	return &toShow, qType
}

func judgeAlarmsByCron(coversationAlarm *map[string][]pkg.AlarmConfig) {
	c := cron.New(
		cron.WithParser(
			cron.NewParser(
				cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)))

	for convTitle, envConf := range *coversationAlarm {
		dingDingTokens, err := GetDingDingTokensByConvTitle(convTitle, pkg.Cfg)
		if dingDingTokens == nil || err != nil {
			logger.WithFields(logger.Fields{
				"convTitle": convTitle,
				"pkg.Cfg":   pkg.Cfg,
			}).Errorf("get dingdingTokens by conv title error: " + err.Error())
			continue
		}
		// new advisor
		for _, conf := range envConf {
			if conf.Cons == nil {
				continue
			}
			if len(conf.Cron) == 0 {
				continue
			}
			cron, cons, priceMax := conf.Cron, conf.Cons, conf.PriceMax
			fmt.Printf("add cron func,cron: %s, cons:%s, dingDingTokens:%s, priceMax:%s", cron, cons, dingDingTokens, priceMax)
			toShow := &[]pkg.AdvisorResponse{}
			showed := &[]pkg.AdvisorChangedInstance{}
			newChanged := &[]pkg.AdvisorResponse{}
			var startTime *int64
			startTime = new(int64)
			*startTime = time.Now().Unix()
			c.AddFunc(cron, func() {
				judgeOne(cons, dingDingTokens, priceMax, toShow, showed, newChanged, startTime)
			})
		}
	}

	fmt.Printf("\ncron start .....\n\n")
	c.Start()
}

func judgeOne(advisor *pkg.Advisor, dingDingTokens *[]string, priceMax float64, nowResp *[]pkg.AdvisorResponse,
	showed *[]pkg.AdvisorChangedInstance, newChanged *[]pkg.AdvisorResponse, lastTime *int64) {
	logger.Infof("advisor is", advisor)

	*newChanged = []pkg.AdvisorResponse{}

	nowResp, _ = getAlarmsByReq(advisor, priceMax)

	if nowResp == nil {
		return
	}

	for _, item := range *nowResp {
		foundRes := isItemExists(item, showed, priceMax)
		if foundRes == 0 || foundRes == 2 { // not exactly same
			*newChanged = append(*newChanged, item)
		}
	}

	if (time.Now().Unix() - *lastTime) >= 24*60*60 {
		*showed = []pkg.AdvisorChangedInstance{}
		for _, item := range *newChanged {
			*showed = append(*showed, pkg.AdvisorChangedInstance{
				InstanceTypeId:   item.InstanceTypeId,
				ZoneId:           item.ZoneId,
				PricePerCore:     item.PricePerCore,
				LastPricePerCore: item.PricePerCore,
			})
		}
		*lastTime = time.Now().Unix()
		fmt.Println("lastTime is set to be", *lastTime)
		fmt.Println("after oneday showed reset to be ", *showed)
	}

	if len(*newChanged) != 0 {
		logger.Infof("*newChanged is: ", *newChanged)

		toShow := make([]pkg.AdvisorResponse, len(*newChanged))
		copy(toShow, *newChanged)
		isSend := SendResToDingDing(advisor, dingDingTokens, &toShow, UnKown, priceMax, true)
		if !isSend {
			logger.Errorf("SendResToDingDing res is false, to show data is ", *newChanged)
		}
	}

}

func isItemExists(toSearch pkg.AdvisorResponse, sets *[]pkg.AdvisorChangedInstance, priceMax float64) int {
	found, setKey := 0, 0
	for key, item := range *sets {
		if toSearch.InstanceTypeId == item.InstanceTypeId && toSearch.ZoneId == item.ZoneId {
			if toSearch.PricePerCore == item.PricePerCore {
				found = 1
				break
			} else {
				setKey = key

				itemFloat, err := strconv.ParseFloat(item.PricePerCore, 64)
				if err != nil {
					logger.Errorf("parse str to float error: ", err.Error())
					continue
				}
				searchFloat, err := strconv.ParseFloat(toSearch.PricePerCore, 64)
				if err != nil {
					logger.Errorf("parse str to float error: ", err.Error())
					found = 1
					break
				}
				if (itemFloat >= priceMax && searchFloat >= priceMax) || (itemFloat < priceMax && searchFloat < priceMax) {
					found = 3
				} else {
					found = 2
				}
			}
		}
	}
	if found == 2 || found == 3 {
		(*sets)[setKey].LastPricePerCore = (*sets)[setKey].PricePerCore
		(*sets)[setKey].PricePerCore = toSearch.PricePerCore
	}

	if found == 3 {
		logger.Warnf("res is 3: ", toSearch)
	}

	if found == 0 {
		// not found
		*sets = append(*sets, pkg.AdvisorChangedInstance{
			InstanceTypeId:   toSearch.InstanceTypeId,
			ZoneId:           toSearch.ZoneId,
			PricePerCore:     toSearch.PricePerCore,
			LastPricePerCore: toSearch.PricePerCore,
		})
		logger.Warnf("append to sets: ", toSearch)
	}
	return found
}

func updateShowed(newChange pkg.AdvisorResponse, showed *[]pkg.AdvisorChangedInstance) {
	for _, item := range *showed {
		if newChange.InstanceTypeId == item.InstanceTypeId && newChange.ZoneId == item.ZoneId && newChange.PricePerCore != item.PricePerCore {
			item.LastPricePerCore = item.PricePerCore
			item.PricePerCore = newChange.PricePerCore
		}
	}
}

func isSameAdvisorResponse(lastResp *[]pkg.AdvisorResponse, nowResp *[]pkg.AdvisorResponse) bool {
	same := false
	if lastResp != nil && nowResp != nil {
		for _, item := range *nowResp {
			found := 0
			for _, last := range *lastResp {
				if item == last {
					found = 1
				}
			}
			if found == 0 {
				fmt.Println("not found", item)
				logger.Infof("not found", item)
				same = false
				break
			}
			same = true
		}
	}
	return same
}
