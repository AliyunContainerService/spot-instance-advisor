package pkg

import (
	"encoding/json"
	"fmt"
	logger "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type ErrorType int
type SenderName string

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

type Markdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type MarkdownMsg struct {
	MsgType  string   `json:"msgtype"`
	Markdown Markdown `json:"markdown"`
}

type DingDingSendor struct {
	SenderConfig
	Qtype  Querytype
	ToSend string
}

// ding ding ai need resp struct
type DingDingRep struct {
	Success   bool              `json:"success"`
	ErrorCode string            `json:"errorCode"`
	ErrorMsg  string            `json:"errorMsg"`
	Fields    map[string]string `json:"fields"`
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

func SendResponse(w http.ResponseWriter, resp interface{}) {
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

type Sender interface {
	SetQueryType(Querytype)
	SetTokens(*[]string) error
	SetToSend(string) bool
	Send()
}

func NewSendor(config *SenderConfig) Sender {
	if config == nil {
		return nil
	}
	switch strings.ToUpper(strings.TrimSpace(string(config.SenderName))) {
	case string(DEFAULTSENDER):
		return &DingDingSendor{
			SenderConfig: *config,
			//DingDingToken: *config.Token,
			ToSend: "",
			Qtype:  OnceQuery,
		}
	default:
		return nil
	}
}

func (ddSendor *DingDingSendor) SetTokens(conversationTitles *[]string) error {
	newTokens := map[string]string{}
	for _, converTitle := range *conversationTitles {
		if token, ok := (*ddSendor.Token)[converTitle]; !ok || len(token) == 0 {
			if cfgToken := getTokenFromCfg(ddSendor.SenderName, converTitle); cfgToken != nil && len(*cfgToken) != 0 {
				newTokens[converTitle] = *cfgToken
			} else {
				return fmt.Errorf("%s's sender token is EMPTY!", converTitle)
			}
		} else {
			newTokens[converTitle] = token
		}
	}
	*ddSendor.Token = newTokens
	logger.Infof("set tokens are", (*ddSendor.Token))
	return nil
}

func getTokenFromCfg(name SenderName, converTitle string) *string {
	for _, senderConfig := range Cfg.DefaultSender {
		if name == senderConfig.SenderName && senderConfig.Token != nil {
			if token, ok := (*senderConfig.Token)[converTitle]; ok {
				return &token
			}
		}
	}
	return nil
}

func (ddSendor *DingDingSendor) SetQueryType(qtype Querytype) {
	ddSendor.Qtype = qtype
}
func (ddSendor *DingDingSendor) SetToSend(toSend string) bool {
	if len(toSend) == 0 {
		return false
	}
	ddSendor.ToSend = toSend
	return true
}
func (md *Markdown) SetMdTitile(qtype Querytype) {
	switch qtype {
	case Alarm:
		md.Title = ALARMTITLE
	case Recover:
		md.Title = RECOVERALARMTITLE
	case Monitor:
		md.Title = MONITORTITLE
	default:
		md.Title = QUERYTITLE
	}
}
func (ddSendor *DingDingSendor) Send() {
	msg := MarkdownMsg{
		MsgType: MarkdownType,
		Markdown: Markdown{
			Title: ALARMTITLE,
			Text:  "无结果",
		},
	}
	msg.Markdown.SetMdTitile(ddSendor.Qtype)

	if len(ddSendor.ToSend) > 0 {
		msg.Markdown.Text = ddSendor.ToSend
	}

	for _, token := range *ddSendor.Token {
		logger.Error("DO SEND:", token)
		//fmt.Println(fmt.Sprintf("SEND msg %++v", msg))
		err := sendDingDing(msg, token)
		if err != nil {
			logger.Error("sendDingDing return error:", err)
			continue
		}
	}
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
	if err != nil || resp.StatusCode != 200 {
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
