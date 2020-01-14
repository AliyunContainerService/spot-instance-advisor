package handler

import (
	"fmt"
	"github.com/IrisIris/spot-instance-advisor/pkg"
	logger "github.com/Sirupsen/logrus"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

var QUERYNOTEMPTYFIELD = []string{"sys.ding.conversationTitle", "region"}

func SpotHandler(w http.ResponseWriter, r *http.Request) {
	vals := r.URL.Query()
	resp := &DingDingRep{
		Success:   true,
		ErrorCode: strconv.Itoa(int(ErrorNone)),
		ErrorMsg:  "",
		Fields:    nil,
	}
	logger.Info("input vals are :", vals)

	// check && get json byte
	jsonBytes, err := InputCommonCheck(w, vals, pkg.Cfg, &QUERYNOTEMPTYFIELD)
	if jsonBytes == nil || err != nil {
		logger.WithFields(logger.Fields{
			"vals":          vals,
			"notEmptyField": QUERYNOTEMPTYFIELD,
			"error":         err,
		}).Errorf("input common check error :")
		resp = PackageResp(false, "input common check error: "+err.Error(), nil)
		sendResponse(w, resp)
		return
	}

	// get new advisor
	advisor, err := pkg.NewAdvisor(jsonBytes)
	if err != nil {
		logger.WithFields(logger.Fields{
			"advisor": *advisor,
			"error":   err,
		}).Errorf("new advisor error :")
		resp = PackageResp(false, "new advisor err: "+err.Error(), nil)
		sendResponse(w, resp)
		return
	}

	DingDingTokens, err := GetDingDingTokens(vals, pkg.Cfg)
	if err != nil {
		resp = PackageResp(false, err.Error(), nil)
		sendResponse(w, resp)
	}

	// check price max set or not
	var priceMax float64
	if _, ok := vals["price_max"]; !ok || len(vals["price_max"]) == 0 || len(vals["price_max"][0]) == 0 {
		priceMax = 0.0
		resp.Fields = map[string]string{
			"title": QUERYTITLE,
		}
		go queryByReq(advisor, DingDingTokens)
	} else {
		if toPriceMax, err := strconv.ParseFloat(vals["price_max"][0], 64); err != nil {
			logger.Errorf("strconv ParseFloat vals price_max error", err.Error())
			resp = PackageResp(false, "strconv ParseFloat vals price_max error: "+err.Error(), nil)
			return
		} else {
			priceMax = toPriceMax
			resp.Fields = map[string]string{
				"title": ALARMTITLE,
			}
		}
		toShow := &[]pkg.AdvisorResponse{}
		showed := &[]pkg.AdvisorChangedInstance{}
		newChanged := &[]pkg.AdvisorResponse{}
		var startTime *int64
		startTime = new(int64)
		*startTime = time.Now().Unix()
		//go judgeOneByCron(advisor, DingDingTokens, priceMax, lastShow, toShow)
		go judgeOne(advisor, DingDingTokens, priceMax, toShow, showed, newChanged, startTime)
	}
	fmt.Println("SpotHandler-priceMax is", priceMax)

	sendResponse(w, resp)
}

func queryByReq(advisor *pkg.Advisor, DingDingTokens *[]string) {
	msg := MarkdownMsg{
		MsgType: MarkdownType,
		Markdown: Markdown{
			Title: QUERYTITLE,
			Text:  "无搜索结果",
		},
	}
	qType := OnceQuery

	spotInstancePrices, err := advisor.SpotPricesAnalysis()
	if err != nil {
		logger.WithFields(logger.Fields{
			"spotInstancePrices": spotInstancePrices,
			"error":              err,
		}).Errorf("get spotInstancePrices error :")
		return
	}

	if len(spotInstancePrices) == 0 {
		for _, token := range *DingDingTokens {
			err = sendDingDing(msg, token)
			if err != nil {
				logger.Error("sendDingDing return error:", err)
				continue
			}
		}
		return
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

	SendResToDingDing(advisor, DingDingTokens, &toShow, qType, 0.0, false)
}

func getByReq(advisor *pkg.Advisor) *[]pkg.AdvisorResponse {
	spotInstancePrices, err := advisor.SpotPricesAnalysis()
	if err != nil {
		logger.WithFields(logger.Fields{
			"spotInstancePrices": spotInstancePrices,
			"error":              err,
		}).Errorf("get spotInstancePrices error :")
		return nil
	}

	if len(spotInstancePrices) == 0 {
		return nil
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
	return &toShow
}
