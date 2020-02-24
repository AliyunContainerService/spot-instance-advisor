package pkg

import (
	"encoding/json"
	"fmt"
	logger "github.com/Sirupsen/logrus"
	"strconv"
	"strings"
)

const (
	ALARMFIELDCOLOR = "#DC143C"
	FOOTERCOLOR     = "#A9A9A9"
)

type Querytype int

const (
	OnceQuery Querytype = iota // 0
	Alarm                      // 1
	Recover                    // 2
	Monitor
	UnKnown
)

// ding ding markdown table
type DDMarkdownTable struct {
	Advisor      *Advisor
	MsgType      Querytype
	HeadTitle    string
	TableContent string
	Foot         string // maybe some annotations or explanations of the alarm job
	ContentPrepare
	Config *AlarmConfig
}

type ContentPrepare struct {
	Larger     *[]interface{}
	Smaller    *[]interface{}
	OriginData *[]interface{}
}

func (ddMDTable *DDMarkdownTable) SetConfig(alarmConf *AlarmConfig) {
	//logger.Infof("patten conf is %++v", alarmConf)
	if alarmConf != nil {
		if ddMDTable.Config == nil {
			ddMDTable.Config = &AlarmConfig{}
		}
		*ddMDTable.Config = *alarmConf
	}
}

func (ddMDTable *DDMarkdownTable) SetMsgType(qType Querytype) {
	if qType != UnKnown {
		ddMDTable.MsgType = qType
	} else {
		switch {
		case ddMDTable.Smaller == nil || len(*ddMDTable.Smaller) == 0:
			ddMDTable.MsgType = Alarm
		case ddMDTable.Larger == nil || len(*ddMDTable.Larger) == 0:
			ddMDTable.MsgType = Recover
		default:
			ddMDTable.MsgType = Monitor
		}
	}
}

func (ddMDTable *DDMarkdownTable) SetHeadTitle() {
	switch ddMDTable.MsgType {
	case Alarm:
		ddMDTable.HeadTitle = ALARMMDTITLE
	case Monitor:
		ddMDTable.HeadTitle = MONITORMDTITLE
	case Recover:
		ddMDTable.HeadTitle = RECOVERMDTITLE
	default:
		ddMDTable.HeadTitle = QUERYMDTITLE
	}
}

// if type is monitor or unknown
func (ddMDTable *DDMarkdownTable) PartLargerAndSmaller() {
	threshold, err := strconv.ParseFloat(ddMDTable.Config.Filter.Threshold, 64)
	if err != nil {
		logger.Errorf(fmt.Sprintf(" parse str %s to float error: %s", ddMDTable.Config.Filter.Threshold, err.Error()))
		return
	}

	for _, item := range *ddMDTable.OriginData {
		transferedToMap, err := TransToMap(item)

		judgedBy := ddMDTable.Config.Filter.JudgedByField
		judgedVal := ""
		if val, ok := (*transferedToMap)[judgedBy]; !ok {
			logger.Errorf(fmt.Sprintf("%++v do not have value to be judged by field %s", item, judgedBy))
			continue
		} else {
			judgedVal = fmt.Sprintf("%v", val)
		}
		judgedValFloat, err := strconv.ParseFloat(judgedVal, 64)
		if err != nil {
			logger.Errorf(fmt.Sprintf("parse str %s to float error: %s", judgedVal, err.Error()))
			continue
		}

		if judgedValFloat >= threshold {
			if ddMDTable.Larger == nil {
				ddMDTable.Larger = &[]interface{}{}
			}
			*ddMDTable.Larger = append(*ddMDTable.Larger, item)
		} else {
			if ddMDTable.Smaller == nil {
				ddMDTable.Smaller = &[]interface{}{}
			}
			*ddMDTable.Smaller = append(*ddMDTable.Smaller, item)
		}
	}
}

func (ddMDTable *DDMarkdownTable) SetTableContent() {
	unSetMsgType := ddMDTable.MsgType
	if unSetMsgType == UnKnown {
		ddMDTable.PartLargerAndSmaller()
		ddMDTable.SetMsgType(ddMDTable.MsgType)
	}
	// get formator
	formatorConfig := FormatorConfig{
		Name: MarkdownType,
		MarkDownFormator: MarkDownFormator{
			Advisor:    ddMDTable.Advisor,
			ShowStruct: new(PriceShow),
			HaveHeader: true,
		},
	}
	// base config
	formator := NewFormator(&formatorConfig)
	logger.Infof("message type is %v", ddMDTable.MsgType)

	if ddMDTable.MsgType == Monitor {
		if unSetMsgType != UnKnown {
			ddMDTable.PartLargerAndSmaller()
		}

		// get smaller part
		formator.SetOriginData(ddMDTable.Smaller)
		smallerMdStr, err := formator.Format()
		if err != nil {
			logger.Error("do format return error:", err.Error())
		}
		if len(smallerMdStr) == 0 {
			logger.Errorf("len smeller is %+v", len(*ddMDTable.Smaller))
			logger.Error("smaller than threshold part markdown string is empty")
		}

		// get larger part
		formator.SetOriginData(ddMDTable.Larger)
		formator.SetHaveHeader(false)
		greaterMdStr, err := formator.Format()
		if err != nil {
			logger.Error("do format return error:", err.Error())
		}
		if len(greaterMdStr) == 0 {
			logger.Error("greater than threshold part markdown string is empty")
		}
		if *ddMDTable.Config.Pattern.Color {
			greaterMdStr = colorMdStr(greaterMdStr, ddMDTable.Larger, ddMDTable.Config.Pattern.ColorField)
		}

		ddMDTable.TableContent = smallerMdStr + greaterMdStr
	} else {
		//formator.SetToShow(ddMDTable.OriginData)
		formator.SetOriginData(ddMDTable.OriginData)
		mdStr, err := formator.Format()
		if err != nil {
			logger.Error("do format return error:", err.Error())
			return
		}
		ddMDTable.TableContent = mdStr
		if ddMDTable.MsgType == Alarm && ddMDTable.Config.Pattern.Color != nil && *ddMDTable.Config.Pattern.Color {
			ddMDTable.TableContent = colorMdStr(mdStr, ddMDTable.OriginData, ddMDTable.Config.Pattern.ColorField)
		}
	}

	// move to elsewhere
	if len(ddMDTable.TableContent) == 0 {
		logger.Error("mdStr empty")
	}
}

// todo chang name to colorSpecificField
func colorMdStr(originStr string, toColor *[]interface{}, colorField string) string {
	if toColor == nil || len(*toColor) == 0 {
		return originStr
	}
	if len(originStr) == 0 {
		logger.Errorf("to color origin str is empty")
		return originStr
	}
	haveReplaced := map[string]int{}
	for _, item := range *toColor {
		// get Id from interface
		transedToMap, err := TransToMap(item)
		if err != nil {
			logger.Errorf(fmt.Sprintf("to_color_item trans to map failed, err is %s", err.Error()))
		}
		// todo get from config
		//colorField := "InstanceTypeId"
		instanceTypeId := ""
		if val, ok := (*transedToMap)[colorField]; !ok {
			logger.Errorf(fmt.Sprintf("%v not have the field %s to color"), *transedToMap, colorField)
			continue
		} else {
			instanceTypeId = fmt.Sprintf("%v", val)
		}

		old := strings.Replace(instanceTypeId, "ecs.", "", 1)
		if _, ok := haveReplaced[old]; ok {
			continue
		}
		new := "<font color=" + ALARMFIELDCOLOR + "> " + old + "</font>"
		originStr = strings.Replace(originStr, old, new, -1)
		haveReplaced[old] = 1
	}
	if len(originStr) == 0 {
		logger.Error("after color Str empty")
	}
	return originStr
}

func (ddMDTable *DDMarkdownTable) SetFoot() {
	advisor := ddMDTable.Advisor
	consToshow := AdvisorChinese{
		Region:    advisor.Region,
		Cpu:       advisor.Cpu,
		Memory:    advisor.Memory,
		MaxCpu:    advisor.MaxCpu,
		MaxMemory: advisor.MaxMemory,
		Cutoff:    advisor.Cutoff,
	}
	// get footer str
	consByte, err := json.Marshal(consToshow)
	if err != nil {
		logger.Errorf("json encode error", err.Error())
		consByte = []byte{}
	}
	consStr := string(consByte)
	consStr = strings.Replace(consStr, "\"", "", -1)
	consStr = strings.TrimLeft(consStr, "{")
	consStr = strings.TrimRight(consStr, "}")

	// get final footer
	footer := ""
	switch ddMDTable.MsgType {
	case Alarm:
		footer = "\n\n ##### <font color=" + FOOTERCOLOR + ">设定阈值：" + fmt.Sprintf("%s", ddMDTable.Config.Filter.Threshold) +
			";\n监控条件：" + consStr + "</font>"
	case Recover:
		footer = "\n\n ##### <font color=" + FOOTERCOLOR + ">设定阈值：" + fmt.Sprintf("%s", ddMDTable.Config.Filter.Threshold) +
			";\n监控条件：" + consStr + "</font>"
	case Monitor:
		footer = "\n\n ##### <font color=" + FOOTERCOLOR + ">设定阈值：" + fmt.Sprintf("%s", ddMDTable.Config.Filter.Threshold) +
			";\n监控条件：" + consStr + "</font>"
	default:
		footer = "\n\n ##### <font color=" + FOOTERCOLOR + ">查询条件：" + consStr + "</font>"
	}
	ddMDTable.Foot = footer
}

type MessageDirector struct {
	Builder MessageBuilder
}

func (md *MessageDirector) Create(config *AlarmConfig) string {
	if config == nil {
		return ""
	}
	return md.Builder.SetConfig(config).SetBody().SetHeader().SetFoot().Build()
}

type MessageBuilder interface {
	SetConfig(*AlarmConfig) MessageBuilder
	SetHeader() MessageBuilder
	SetBody() MessageBuilder
	SetFoot() MessageBuilder
	Build() string
}

type DDMarkdownTableBuilder struct {
	DDMarkdownTable *DDMarkdownTable
}

func (d *DDMarkdownTableBuilder) SetConfig(config *AlarmConfig) MessageBuilder {
	if config == nil {

	}
	if d.DDMarkdownTable == nil {
		d.DDMarkdownTable = &DDMarkdownTable{}
	}
	d.DDMarkdownTable.SetConfig(config)
	return d
}

func (d *DDMarkdownTableBuilder) SetHeader() MessageBuilder {
	if d.DDMarkdownTable == nil {
		d.DDMarkdownTable = &DDMarkdownTable{}
	}
	d.DDMarkdownTable.SetHeadTitle()
	return d
}

func (d *DDMarkdownTableBuilder) SetBody() MessageBuilder {
	if d.DDMarkdownTable == nil {
		d.DDMarkdownTable = &DDMarkdownTable{}
	}
	d.DDMarkdownTable.SetTableContent()
	return d
}

func (d *DDMarkdownTableBuilder) SetFoot() MessageBuilder {
	if d.DDMarkdownTable == nil {
		d.DDMarkdownTable = &DDMarkdownTable{}
	}
	d.DDMarkdownTable.SetFoot()
	return d
}

func (d *DDMarkdownTableBuilder) Build() string {
	return d.DDMarkdownTable.HeadTitle + "\n\n" + d.DDMarkdownTable.TableContent + "\n\n" + d.DDMarkdownTable.Foot
}
