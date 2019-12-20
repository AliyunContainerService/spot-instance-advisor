package main

import (
	"encoding/json"
	"fmt"
	"github.com/AliyunContainerService/spot-instance-advisor/pkg"
	logger "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	MarkdownType   = "markdown"
	PriceTitle     = "Prices List"
	DINGBOTWEBHOOK = "https://oapi.dingtalk.com/robot/send?access_token="
)

type ErrorType int

const (
	ErrorNone ErrorType = http.StatusOK
)

type MarkdownMsg struct {
	MsgType  string   `json:"msgtype"`
	Markdown Markdown `json:"markdown"`
}

type Markdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}
type DingDingRep struct {
	Success   bool              `json:"success"`
	ErrorCode string            `json:"errorCode"`
	ErrorMsg  string            `json:"errorMsg"`
	Fields    map[string]string `json:"fields"`
}

var cfg *pkg.Config

func init() {
	cfg = pkg.LoadConfig()
	pkg.ConfigLogger(cfg.LogLevel)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/spot", spotHandler)
	http.Handle("/", r)

	srv := &http.Server{
		Handler: r,
		Addr:    ":8000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func spotHandler(w http.ResponseWriter, r *http.Request) {
	vals := r.URL.Query()
	resp := &DingDingRep{
		Success:   true,
		ErrorCode: strconv.Itoa(int(ErrorNone)),
		ErrorMsg:  "",
		Fields:    nil,
	}
	logger.Info("input vals are :", vals)

	// check query params
	if item, ok := vals["sys.ding.conversationTitle"]; !ok || len(item) == 0 {
		resp.ErrorMsg = "EMPTY sys.ding.conversationTitle!"
		sendResponse(w, resp)
		return
	}
	var DingDingTokens []string
	for _, name := range vals["sys.ding.conversationTitle"] {
		if token, ok := cfg.DingDingTokensMap[name]; !ok || len(token) == 0 {
			resp.ErrorMsg = name + "'s DingDingToken  EMPTY!"
			sendResponse(w, resp)
			return
		} else {
			DingDingTokens = append(DingDingTokens, token)
		}
	}
	// set default
	if v, ok := vals["access_key_id"]; !ok || len(v) == 0 || len(v[0]) == 0 {
		vals["access_key_id"] = []string{cfg.AccessKeyId}
	}
	if v, ok := vals["access_key_secret"]; !ok || len(v) == 0 || len(v[0]) == 0 {
		vals["access_key_secret"] = []string{cfg.AccessKeySecret}
	}

	jsonBytes, err := transReqToJson(vals)
	if err != nil {
		resp.Success = false
		resp.ErrorMsg = err.Error()
		sendResponse(w, resp)
		return
	}
	// get new advisor
	advisor, err := pkg.NewAdvisor(jsonBytes)
	if err != nil {
		logger.WithFields(logger.Fields{
			"advisor": *advisor,
			"error":   err,
		}).Infof("new advisor error :")
		resp.Success = false
		resp.ErrorMsg = "new advisor err: " + err.Error()
		sendResponse(w, resp)
		return
	}

	go queryByReq(advisor, DingDingTokens)

	resp.Fields = map[string]string{
		"title": PriceTitle,
	}
	sendResponse(w, resp)
}

func queryByReq(advisor *pkg.Advisor, DingDingTokens []string) {
	msg := MarkdownMsg{
		MsgType: MarkdownType,
		Markdown: Markdown{
			Title: PriceTitle,
			Text:  "无搜索结果",
		},
	}
	spotInstancePrices, err := advisor.SpotPricesAnalysis()
	if err != nil {
		logger.WithFields(logger.Fields{
			"spotInstancePrices": spotInstancePrices,
			"error":              err,
		}).Debug("get spotInstancePrices error :")
		return
	}

	if len(spotInstancePrices) == 0 {
		for _, token := range DingDingTokens {
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

	mdStr, err := pkg.DoFormat(MarkdownType, &toShow)
	if err != nil {
		logger.Error("do format return error:", err.Error())
		return
	}
	if len(mdStr) == 0 {
		logger.Error("mdStr empty")
		for _, token := range DingDingTokens {
			err = sendDingDing(msg, token)
			if err != nil {
				logger.Error("sendDingDing return error:", err)
				continue
			}
		}
		return
	}

	msg.Markdown.Text = mdStr
	for _, token := range DingDingTokens {
		err = sendDingDing(msg, token)
		if err != nil {
			logger.Error("sendDingDing return error:", err)
			continue
		}
	}
}

func sendResponse(w http.ResponseWriter, resp *DingDingRep) {
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	w.Write(jsonBytes)
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
