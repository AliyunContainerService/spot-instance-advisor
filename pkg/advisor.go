package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	ecsService "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
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
	AccessKeyId     string `json:"access_key_id,omitempty"`
	AccessKeySecret string `json:"access_key_secret,omitempty"`
	Region          string `json:"region"`
	Cpu             int    `json:"cpu"`
	Memory          int    `json:"memory"`
	MaxCpu          int    `json:"max_cpu"`
	MaxMemory       int    `json:"max_memory"`
	Family          string `json:"family,omitempty"`
	Cutoff          int    `json:"cutoff"`
	Limit           int    `json:"limit"`
	Resolution      int    `json:"resolution"`
}

type ChAdvisor struct {
	Region    string `json:"地域"`
	Cpu       int    `json:"cpu最小值"`
	Memory    int    `json:"memory最小值"`
	MaxCpu    int    `json:"cpu最大值"`
	MaxMemory int    `json:"memory最大值"`
	Cutoff    int    `json:"折扣"`
}

type AdvisorResponse struct {
	InstanceTypeId string
	ZoneId         string
	PricePerCore   string
}

type AdvisorChangedInstance struct {
	InstanceTypeId   string
	ZoneId           string
	PricePerCore     string
	LastPricePerCore string
}

func NewAdvisor(reqJson []byte) (*Advisor, error) {
	advisor := &Advisor{
		AccessKeyId:     "",
		AccessKeySecret: "",
		Region:          "",
		Cpu:             DEFAULTCPU,
		Memory:          2,
		MaxCpu:          DEFAULTMAXCPU,
		MaxMemory:       64,
		Family:          DEFAULTFAMILY,
		Cutoff:          2,
		Limit:           DEFAULTLIMIT,
		Resolution:      7,
	}

	if len(reqJson) == 0 {
		return advisor, nil
		//return nil, fmt.Errorf("cannot be empty")
	}

	err := json.Unmarshal(reqJson, advisor)
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
