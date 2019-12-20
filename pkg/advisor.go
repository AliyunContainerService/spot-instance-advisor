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

type AdvisorResponse struct {
	InstanceTypeId string
	ZoneId         string
	PricePerCore   string
}

func NewAdvisor(reqJson []byte) (*Advisor, error) {
	if len(reqJson) == 0 {
		return nil, fmt.Errorf("cannot be empty")
	}

	advisor := &Advisor{
		AccessKeyId:     "",
		AccessKeySecret: "",
		Region:          "",
		Cpu:             1,
		Memory:          2,
		MaxCpu:          32,
		MaxMemory:       64,
		Family:          "",
		Cutoff:          2,
		Limit:           20,
		Resolution:      7,
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
