package handler

import (
	"github.com/IrisIris/spot-instance-advisor/pkg"
	logger "github.com/Sirupsen/logrus"
	"net/http"
	"sort"
	"strconv"
)

var QUERYNOTEMPTYFIELD = []string{"sys.ding.conversationTitle", "region"}

func SpotHandler(w http.ResponseWriter, r *http.Request) {
	vals := r.URL.Query()
	resp := &pkg.DingDingRep{
		Success:   true,
		ErrorCode: strconv.Itoa(int(pkg.ErrorNone)),
		ErrorMsg:  "",
		Fields:    nil,
	}
	logger.Info("input query values are :", vals)

	// get new advisor
	advisor, err := pkg.NewAdisorByReq(&vals, pkg.Cfg, &QUERYNOTEMPTYFIELD)
	if err != nil {
		logger.WithFields(logger.Fields{
			"input_val": vals,
			"error":     err,
		}).Errorf("new advisor error :")
		resp = pkg.PackageResp(false, "new advisor err: "+err.Error(), nil)
		pkg.SendResponse(w, resp)
		return
	}

	// check threshold set or not, if set is similar to threshold
	if _, ok := vals["threshold"]; !ok || len(vals["threshold"]) == 0 || len(vals["threshold"][0]) == 0 {
		resp.Fields = map[string]string{
			"title": pkg.QUERYTITLE,
		}
		converTitles := vals["sys.ding.conversationTitle"]
		go queryByReq(advisor, &converTitles)
	} else {
		if _, err := strconv.ParseFloat(vals["threshold"][0], 64); err != nil {
			logger.Errorf("parse  threshold from string to float  error", err.Error())
			resp = pkg.PackageResp(false, "parse  threshold from string to float  error: "+err.Error(), nil)
			pkg.SendResponse(w, resp)
			return
		}
		resp.Fields = map[string]string{
			"title": pkg.ALARMTITLE,
		}
		filter := pkg.FilterConfig{
			JudgedByField: pkg.Cfg.DefaultFilter.JudgedByField,
			Threshold:     vals["threshold"][0],
		}
		if _, ok := vals["judge"]; ok && len(vals["judge"]) != 0 || len(vals["judge"][0]) != 0 {
			filter.JudgedByField = vals["judge"][0]
		}

		alarmConfig := &pkg.AlarmConfig{
			Filter:  &filter,
			Sender:  pkg.Cfg.DefaultSender,
			Pattern: pkg.Cfg.DefaultPattern,
		}

		showed := make((map[string]*pkg.ChangedOne))
		alarmRecord := AlarmJobRecord{
			NowResp:    &[]pkg.InstancePrice{},
			Showed:     &showed,
			NewChanged: &[]pkg.InstancePrice{},
		}
		for _, convTitle := range vals["sys.ding.conversationTitle"] {
			go alarmJob(advisor, convTitle, alarmConfig, alarmRecord)
		}
	}

	pkg.SendResponse(w, resp)
}

func queryByReq(advisor *pkg.Advisor, converTitles *[]string) {
	showStr := ""
	spotInstancePrices, err := advisor.SpotPricesAnalysis()
	if err != nil {
		logger.WithFields(logger.Fields{
			"spotInstancePrices": spotInstancePrices,
			"error":              err,
		}).Errorf("get spotInstancePrices error :")
		return
	}
	if len(spotInstancePrices) != 0 {
		sort.Sort(spotInstancePrices)
		// change to []interface{}
		interfaceSlice := make([]interface{}, len(spotInstancePrices))
		for i, d := range spotInstancePrices {
			interfaceSlice[i] = d
		}

		var builder = &pkg.DDMarkdownTableBuilder{}
		builder.DDMarkdownTable = &pkg.DDMarkdownTable{
			Advisor:        advisor,
			MsgType:        pkg.OnceQuery,
			ContentPrepare: pkg.ContentPrepare{OriginData: &interfaceSlice},
			Config: &pkg.AlarmConfig{
				Filter:  pkg.Cfg.DefaultFilter,
				Sender:  pkg.Cfg.DefaultSender,
				Pattern: pkg.Cfg.DefaultPattern,
			},
		}
		director := &pkg.MessageDirector{builder}
		showStr = director.Create(builder.DDMarkdownTable.Config)
	}

	for _, senderConfig := range pkg.Cfg.DefaultSender {
		logger.Errorf("converTitles is ", converTitles)
		sender := pkg.NewSendor(senderConfig)
		if sender == nil {
			logger.WithField("sender_config", senderConfig).Error("get sender is nil, maybe unknown sender name")
			return
		}
		err := sender.SetTokens(converTitles)
		if err != nil {
			logger.Errorf("set sender tokens failed:", err)
			return
		}

		sender.SetQueryType(pkg.OnceQuery)
		sender.SetToSend(showStr)
		sender.Send()
	}
}
