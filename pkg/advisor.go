package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	logger "github.com/Sirupsen/logrus"
	ecsService "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"net/url"
	"sort"
)

var (
	NOTEMPTYFIELD = []string{"access_key_id", "access_key_secret"}
)

const (
	DEFAULTFAMILY = "ecs.ec1,ecs.sn1ne,ecs.c5,ecs.c6"
	DEFAULTCPU    = 4
	DEFAULTMAXCPU = 128
	DEFAULTLIMIT  = 50
)

type Advisor struct {
	AccessKeyId     string  `json:"access_key_id,omitempty"`
	AccessKeySecret string  `json:"access_key_secret,omitempty"`
	Region          string  `json:"region"`
	Cpu             int     `json:"cpu"`
	Memory          int     `json:"memory"`
	MaxCpu          int     `json:"max_cpu"`
	MaxMemory       int     `json:"max_memory"`
	Family          string  `json:"family,omitempty"`
	Cutoff          float64 `json:"cutoff"`
	Limit           int     `json:"limit"`
	Resolution      int     `json:"resolution"`
}

type AdvisorChinese struct {
	Region    string  `json:"地域"`
	Cpu       int     `json:"cpu最小值"`
	Memory    int     `json:"memory最小值"`
	MaxCpu    int     `json:"cpu最大值"`
	MaxMemory int     `json:"memory最大值"`
	Cutoff    float64 `json:"折扣"`
}

func NewAdisorByReq(vals *url.Values, cfg *Config, notEmptyField *[]string) (*Advisor, error) {
	if vals == nil {
		return NewAdvisor(nil)
	}

	jsonBytes, err := TransToJsonBytes(vals, cfg, notEmptyField)
	if jsonBytes == nil || err != nil {
		if notEmptyField == nil {
			notEmptyField = &[]string{"nil"}
		}
		logger.WithFields(logger.Fields{"to_trans_json_data": *vals, "not_empty_field": *notEmptyField, "error": err}).
			Errorf("trans to advisor json bytes error:")
		return nil, err
	}
	return NewAdvisor(&jsonBytes)
}

func NewAdvisor(reqJson *[]byte) (*Advisor, error) {
	advisor := &Advisor{
		AccessKeyId:     "",
		AccessKeySecret: "",
		Region:          "",
		Cpu:             DEFAULTCPU,
		Memory:          2,
		MaxCpu:          DEFAULTMAXCPU,
		MaxMemory:       64,
		Family:          DEFAULTFAMILY,
		Cutoff:          2.0,
		Limit:           DEFAULTLIMIT,
		Resolution:      7,
	}

	if reqJson == nil || len(*reqJson) == 0 {
		return advisor, nil
	}

	err := json.Unmarshal(*reqJson, advisor)
	if err != nil {
		return nil, err
	}
	return advisor, nil
}

func (req *Advisor) SpotPricesAnalysis() (SortedInstancePrices, error) {
	client, err := ecsService.NewClientWithAccessKey(req.Region, req.AccessKeyId, req.AccessKeySecret)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to create ecs client,because:  %s", err.Error()))
		// panic(fmt.Sprintf("Failed to create ecs client,because of %s", err.Error()))
	}

	metastore := NewMetaStore(client)

	err = metastore.Initialize(req.Region)
	if err != nil {
		return nil, err
	}

	instanceTypes := metastore.FilterInstances(req.Cpu, req.Memory, req.MaxCpu, req.MaxMemory, req.Family)

	historyPrices := metastore.FetchSpotPrices(instanceTypes, req.Resolution)

	spotInstancePrices := metastore.SpotPricesAnalysis(historyPrices)

	return spotInstancePrices, nil
}

func (advisor *Advisor) GetAnalysisRes() *[]InstancePrice {
	// get current value
	if advisor == nil {
		logger.Error("get advisor response input error: advisor is nil")
		return nil
	}
	spotInstancePrices, err := advisor.SpotPricesAnalysis()
	if err != nil {
		logger.WithFields(logger.Fields{
			"spotInstancePrices": spotInstancePrices,
			"error":              err,
		}).Errorf("get spotInstancePrices error :")
		return nil
	}

	if len(spotInstancePrices) == 0 {
		logger.WithFields(logger.Fields{
			"spotInstancePrices": spotInstancePrices,
			"error":              nil,
		}).Errorf("spot prices analysis get empty spotInstancePrices :")
		return nil
	}
	sort.Sort(spotInstancePrices)

	resp := []InstancePrice{}
	count := 0
	for _, item := range spotInstancePrices {
		if count >= advisor.Limit {
			break
		}
		resp = append(resp, item)
		count++
	}
	return &resp
}
