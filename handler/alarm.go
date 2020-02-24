package handler

import (
	"fmt"
	"github.com/IrisIris/spot-instance-advisor/pkg"
	logger "github.com/Sirupsen/logrus"
	"github.com/robfig/cron"
	"net/http"
	"net/url"
)

type AlarmJobRecord struct {
	NowResp    *[]pkg.InstancePrice
	Showed     *map[string]*pkg.ChangedOne
	NewChanged *[]pkg.InstancePrice
}

func AlarmHandler(w http.ResponseWriter, r *http.Request) {
	vals := url.Values{}
	// get advisor && set default val
	advisor, err := pkg.NewAdisorByReq(&vals, pkg.Cfg, nil)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("new advisor error :")
		resp := pkg.PackageResp(false, "new advisor err: "+err.Error(), nil)
		pkg.SendResponse(w, resp)
		return
	}
	fmt.Println("alarm init advisor is", advisor)
	logger.Info("alarm init advisor is", advisor)

	// update advisor from con && parse alarm related config from env
	envAlarmConfig, err := pkg.LoadAlarmConfig(advisor)
	if err != nil {
		logger.Errorf("load alarm config error: " + err.Error())
		resp := pkg.PackageResp(false, "load alarm config error: "+err.Error(), nil)
		pkg.SendResponse(w, resp)
		return
	}

	// do cron job
	alarmsByCron(envAlarmConfig)
	resp := pkg.PackageResp(true, "", &map[string]string{"title": pkg.ALARMTITLE})
	pkg.SendResponse(w, resp)
}

func alarmsByCron(alarmConvMap pkg.EnvAlarmConfig) {
	parser := cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	c := cron.New(cron.WithParser(parser))

	for convTitle, alarmConfs := range *alarmConvMap {
		for _, conf := range alarmConfs {
			if conf.Cons == nil {
				logger.Info("alarm condition is empty:", convTitle)
				continue
			}
			_, err := parser.Parse(conf.Cron)
			if err != nil {
				logger.WithFields(logger.Fields{
					"cron_str": conf.Cron,
					"error":    err,
				}).Errorf("parse cron_str error :")
				continue
			}

			showed := make(map[string]*pkg.ChangedOne)
			alarmRecord := AlarmJobRecord{
				NowResp:    &[]pkg.InstancePrice{},
				Showed:     &showed,
				NewChanged: &[]pkg.InstancePrice{},
			}
			cron, cons := conf.Cron, conf.Cons
			fmt.Printf("add to cron func,cron: %s, cons:%s, convTitle:%s\n\n", cron, cons, convTitle)
			jobId, err := c.AddFunc(cron, func() {
				alarmJob(cons, convTitle, &conf, alarmRecord)
			})
			if err != nil {
				logger.WithFields(logger.Fields{"cron": cron, "cons": cons, "conver_title": convTitle, "err": err}).Errorf("add to cron func error:")
				continue
			}
			logger.WithFields(logger.Fields{"cron": cron, "cons": cons, "conver_title": convTitle, "job_id": jobId}).Infof("add to cron func:")
		}
	}

	fmt.Println("\ncron start .....")
	c.Start()
}

func alarmJob(advisor *pkg.Advisor, convTitle string, alarmConfig *pkg.AlarmConfig, alarmRecord AlarmJobRecord) {
	*alarmRecord.NewChanged = []pkg.InstancePrice{}

	sortedInstancePrices := advisor.GetAnalysisRes()
	if sortedInstancePrices == nil {
		logger.WithFields(logger.Fields{"advisor": advisor}).Infof("get empty advisor analysis resp:")
		return
	}
	logger.Debug("one of  advisor analysis resp is", (*sortedInstancePrices)[0])

	// filter sorted res to store in new changed slice
	filter := pkg.NewFilter(alarmConfig.Filter)
	if filter == nil {
		logger.WithFields(logger.Fields{"filter_config": *alarmConfig.Filter}).Error("new filter is nil")
		return
	}
	toFilterData := pkg.FilterData{
		Currents: sortedInstancePrices,
		Showed:   alarmRecord.Showed,
	}
	logger.WithFields(logger.Fields{"current_to_filter": *sortedInstancePrices, "showed": *alarmRecord.Showed}).
		Debug("to filter data are")
	if filteredData := filter.Filt(&toFilterData); filteredData != nil {
		*alarmRecord.NewChanged = *filteredData
	}

	if len(*alarmRecord.NewChanged) != 0 {
		logger.Debug("after filter and get new changed are", *alarmRecord.NewChanged)
		toShow := make([]pkg.InstancePrice, len(*alarmRecord.NewChanged))
		copy(toShow, *alarmRecord.NewChanged)
		logger.Infof("len toshow is", len(toShow))

		// build message depend on new changed data
		builder := &pkg.DDMarkdownTableBuilder{}
		builder.DDMarkdownTable = &pkg.DDMarkdownTable{
			Advisor:        advisor,
			MsgType:        pkg.UnKnown,
			ContentPrepare: pkg.ContentPrepare{OriginData: pkg.TransPriceSlice2ISlice(&toShow)},
			Config:         alarmConfig,
		}
		director := &pkg.MessageDirector{Builder: builder}
		DDMarkdownTableStr := director.Create(alarmConfig)
		// to send message
		for _, senderConfig := range alarmConfig.Sender {
			sender := pkg.NewSendor(senderConfig)
			if sender == nil {
				logger.WithField("sender_config", senderConfig).Error("get sender is nil, maybe unknown sender name")
				return
			}
			err := sender.SetTokens(&[]string{convTitle})
			if err != nil {
				logger.Errorf("set sender tokens failed:", err)
				return
			}
			sender.SetQueryType(builder.DDMarkdownTable.MsgType)
			sender.SetToSend(DDMarkdownTableStr)
			sender.Send()
		}
	}
}
