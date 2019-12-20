package handler

import (
	"encoding/json"
	"fmt"
	"github.com/IrisIris/spot-instance-advisor/pkg"
	logger "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	ErrorNone         ErrorType = http.StatusOK
	MarkdownType                = "markdown"
	DINGBOTWEBHOOK              = "https://oapi.dingtalk.com/robot/send?access_token="
	QUERYTITLE                  = "查询消息"
	ALARMTITLE                  = "报警消息"
	RECOVERALARMTITLE           = "恢复提醒"
	MONITORTITLE                = "监控提醒"
	QUERYMDTITLE                = "### <font color=#1E90FF size=4 >查询结果</font> \n\n"
	MONITORMDTITLE              = "### <font color=#33CCFF size=4 >监控提醒</font> \n\n"
	ALARMMDTITLE                = "### <font color=#B22222 size=4 >超过阈值</font> \n\n"
	RECOVERMDTITLE              = "### <font color=#228B22 size=4 >恢复</font> \n\n"
)

type Querytype int

const (
	OnceQuery Querytype = iota // 0
	Alarm                      // 1
	Recover                    // 2
	Monitor
	UnKown
)

// common used in handlers
// resp
type DingDingRep struct {
	Success   bool              `json:"success"`
	ErrorCode string            `json:"errorCode"`
	ErrorMsg  string            `json:"errorMsg"`
	Fields    map[string]string `json:"fields"`
}

type ErrorType int

type MarkdownMsg struct {
	MsgType  string   `json:"msgtype"`
	Markdown Markdown `json:"markdown"`
}

type Markdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

func InputCommonCheck(w http.ResponseWriter, vals url.Values, cfg *pkg.Config, notEmptyField *[]string) ([]byte, error) {
	// check not empty exists
	if notEmptyField != nil {
		for _, field := range *notEmptyField {
			if item, ok := vals[field]; !ok || len(item) == 0 {
				resp := PackageResp(false, "EMPTY "+field, nil)
				sendResponse(w, resp)
				return nil, fmt.Errorf("EMPTY " + field)
			}
		}
	}

	// set default val
	if v, ok := vals["access_key_id"]; !ok || len(v) == 0 || len(v[0]) == 0 {
		vals["access_key_id"] = []string{cfg.AccessKeyId}
	}
	if v, ok := vals["access_key_secret"]; !ok || len(v) == 0 || len(v[0]) == 0 {
		vals["access_key_secret"] = []string{cfg.AccessKeySecret}
	}
	jsonBytes, err := transReqToJson(vals)
	if err != nil {
		resp := PackageResp(false, err.Error(), nil)
		sendResponse(w, resp)
		return nil, err
	}
	return jsonBytes, nil
}

func GetDingDingTokens(vals map[string][]string, cfg *pkg.Config) (*[]string, error) {
	var DingDingTokens []string
	for _, name := range vals["sys.ding.conversationTitle"] {
		if token, ok := cfg.DingDingTokensMap[name]; !ok || len(token) == 0 {
			return nil, fmt.Errorf("%s's DingDingToken  EMPTY!", name)
		} else {
			DingDingTokens = append(DingDingTokens, token)
		}
	}
	return &DingDingTokens, nil
}

func GetDingDingTokensByConvTitle(converTitle string, cfg *pkg.Config) (*[]string, error) {
	if len(converTitle) == 0 || cfg == nil {
		return nil, fmt.Errorf("empty input")
	}
	var DingDingTokens []string
	if token, ok := cfg.DingDingTokensMap[converTitle]; !ok || len(token) == 0 {
		return nil, fmt.Errorf("%s's DingDingToken  EMPTY!", converTitle)
	} else {
		DingDingTokens = append(DingDingTokens, token)
	}
	return &DingDingTokens, nil
}

func sendResponse(w http.ResponseWriter, resp interface{}) {
	if w == nil {
		return
	}
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	w.Write(jsonBytes)
}

func PackageResp(success bool, errMsg string, fields *map[string]string) *DingDingRep {
	resp := &DingDingRep{
		Success:   success,
		ErrorCode: strconv.Itoa(int(ErrorNone)),
		ErrorMsg:  errMsg,
	}
	if fields != nil {
		resp.Fields = *fields
	}
	return resp
}

func colorMdStr(originStr string, toColor *[]pkg.AdvisorResponse) string {
	if toColor == nil || len(*toColor) == 0 {
		return originStr
	}
	if len(originStr) == 0 {
		logger.Errorf("to color origin str is empty")
		return originStr
	}
	haveReplaced := map[string]int{}
	for _, item := range *toColor {
		old := strings.Replace(item.InstanceTypeId, "ecs.", "", 1)
		if _, ok := haveReplaced[old]; ok {
			continue
		}
		new := "<font color=#DC143C> " + old + "</font>"
		originStr = strings.Replace(originStr, old, new, -1)
		haveReplaced[old] = 1
	}
	if len(originStr) == 0 {
		logger.Error("after color Str empty")
	}
	return originStr
}

func SendResToDingDing(advisor *pkg.Advisor, dingDingTokens *[]string, toShow *[]pkg.AdvisorResponse, qType Querytype, priceMax float64, color bool) bool {
	if toShow == nil || len(*toShow) == 0 {
		logger.Error("sendResToDingDing input empty toShow is:", toShow)
		return false
	}

	var err error
	msg := MarkdownMsg{
		MsgType: MarkdownType,
		Markdown: Markdown{
			Title: ALARMTITLE,
			Text:  "无结果",
		},
	}
	greater := []pkg.AdvisorResponse{}
	smaller := []pkg.AdvisorResponse{}
	mdStr := ""

	if qType == UnKown {
		for _, item := range *toShow {
			price, err := strconv.ParseFloat(item.PricePerCore, 64)
			if err != nil {
				logger.Errorf(" parse str to float error: ", err.Error())
				continue
			}
			if price >= priceMax {
				greater = append(greater, item)
			} else {
				smaller = append(smaller, item)
			}
		}
		switch {
		case len(smaller) == 0:
			qType = Alarm
		case len(greater) == 0:
			qType = Recover
		default:
			qType = Monitor
		}
	}
	if qType == Monitor {
		smallerMdStr, err := pkg.DoFormat(MarkdownType, &smaller, true, toShow)
		if err != nil {
			logger.Error("do format return error:", err.Error())
			return false
		}
		if len(smallerMdStr) == 0 {
			logger.Error("smallerMdStr empty")
			logger.Errorf("smaller is", smaller)

		}
		greaterMdStr, err := pkg.DoFormat(MarkdownType, &greater, false, toShow)
		if err != nil {
			logger.Error("do format return error:", err.Error())
			return false
		}
		if len(greaterMdStr) == 0 {
			logger.Error("greaterMdStr empty")
			logger.Errorf("greater is", greater)
		}
		if color {
			greaterMdStr = colorMdStr(greaterMdStr, &greater)
		}
		mdStr = smallerMdStr + greaterMdStr
	} else {
		mdStr, err = pkg.DoFormat(MarkdownType, toShow, true, toShow)
		if err != nil {
			logger.Error("do format return error:", err.Error())
			return false
		}
		if qType == Alarm && color {
			mdStr = colorMdStr(mdStr, toShow)
		}
	}

	if len(mdStr) == 0 {
		logger.Error("mdStr empty")
		logger.Errorf("toShow is", toShow)
		msg.Markdown.Text = "mdStr empty"
		return false
	}

	consToshow := pkg.ChAdvisor{
		Region:    advisor.Region,
		Cpu:       advisor.Cpu,
		Memory:    advisor.Memory,
		MaxCpu:    advisor.MaxCpu,
		MaxMemory: advisor.MaxMemory,
		Cutoff:    advisor.Cutoff,
	}
	consByte, err := json.Marshal(consToshow)
	if err != nil {
		logger.Errorf("json encode error", err.Error())
		consByte = []byte{}
	}
	consStr := string(consByte)
	consStr = strings.Replace(consStr, "\"", "", -1)
	consStr = strings.TrimLeft(consStr, "{")
	consStr = strings.TrimRight(consStr, "}")

	switch qType {
	case Alarm:
		msg.Markdown.Text = ALARMMDTITLE
		msg.Markdown.Title = ALARMTITLE
		msg.Markdown.Text += mdStr + "\n\n ##### <font color=#A9A9A9>设定阈值：" + fmt.Sprintf("%.5f", priceMax) +
			";\n监控条件：" + consStr + "</font>"
	case Recover:
		msg.Markdown.Text = RECOVERMDTITLE
		msg.Markdown.Title = RECOVERALARMTITLE
		msg.Markdown.Text += mdStr + "\n\n ##### <font color=#A9A9A9>设定阈值：" + fmt.Sprintf("%.5f", priceMax) +
			";\n监控条件：" + consStr + "</font>"
	case Monitor:
		msg.Markdown.Text = MONITORMDTITLE
		msg.Markdown.Title = MONITORTITLE
		msg.Markdown.Text += mdStr + "\n\n ##### <font color=#A9A9A9>设定阈值：" + fmt.Sprintf("%.5f", priceMax) +
			";\n监控条件：" + consStr + "</font>"
	default:
		msg.Markdown.Text = QUERYMDTITLE
		msg.Markdown.Title = QUERYTITLE
		msg.Markdown.Text += mdStr + "\n\n ##### <font color=#A9A9A9>查询条件：" + consStr + "</font>"
	}

	for _, token := range *dingDingTokens {
		logger.Error("DO SEND")
		err = sendDingDing(msg, token)
		if err != nil {
			logger.Error("sendDingDing return error:", err)
			continue
		}
	}
	return true
}

func sendDingDing(msg MarkdownMsg, dingdingToken string) error {
	reqUrl := DINGBOTWEBHOOK + dingdingToken
	msgJson, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	body := strings.NewReader(string(msgJson))
	req, err := http.NewRequest("POST", reqUrl, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.WithFields(logger.Fields{
			"url":          reqUrl,
			"request_body": string(msgJson),
			"err":          err,
		}).Errorf("request dingding api error: ")
		return err
	}
	if resp.StatusCode != 200 {
		logger.WithFields(logger.Fields{
			"url":          reqUrl,
			"request_body": string(msgJson),
			"status":       resp.Status,
			"status_code":  resp.StatusCode,
		}).Errorf("request dingding api error,invalid http status code: ")
	}

	rsBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.WithFields(logger.Fields{
			"url":           reqUrl,
			"request_body":  string(msgJson),
			"response_body": string(rsBody),
			"resp_body":     resp.Body,
			"status_code":   resp.StatusCode,
		}).Errorf("read dingding api resp error:")
		return err
	}
	defer resp.Body.Close()
	return nil
}

func transReqToJson(req map[string][]string) ([]byte, error) {
	if len(req) == 0 {
		return []byte{}, fmt.Errorf("input cannot be empty")
	}
	toTrans := make(map[string]string, len(req))

	for _, k := range pkg.NOTEMPTYFIELD {
		if val, ok := req[k]; !ok || len(val) == 0 || len(val[0]) == 0 {
			return nil, fmt.Errorf("%s missing", k)
		} else {
			if len(val) == 0 {
				return nil, fmt.Errorf("%s value missing", k)
			} else {
				toTrans[k] = val[0]
			}
		}
	}
	for k, v := range req {
		if len(v) == 0 {
			toTrans[k] = ""
		} else {
			toTrans[k] = v[0]
		}
	}

	bytes, err := json.Marshal(toTrans)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
