package pkg

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	ecsService "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"math"
	"time"
)

const (
	TimestampFormat = "2019-11-20T06:00:00Z"
)

// data structure of instance prices
type InstancePrice struct {
	ecsService.InstanceType
	ZoneId       string
	PricePerCore float64
	Price        string
	Discount     float64
	Possibility  float64
}

// sorted structure of
type SortedInstancePrices []InstancePrice

func (sp SortedInstancePrices) Len() int {
	return len(sp)
}

func (sp SortedInstancePrices) Less(i, j int) bool {

	if sp[i].PricePerCore < sp[j].PricePerCore {
		return true
	}

	//////if sp[i].Possibility < sp[j].Possibility {
	////	return true
	//}

	//if sp[i].Discount < sp[j].Discount {
	//	return true
	//}

	return false
}

func (sp SortedInstancePrices) Swap(i, j int) {
	sp[i], sp[j] = sp[j], sp[i]
}

func CreateInstancePrice(meta ecsService.InstanceType, zoneId string, prices []ecsService.SpotPriceType) InstancePrice {
	latestPrice := FindLatestPrice(prices)
	ip := InstancePrice{
		InstanceType: meta,
		ZoneId:       zoneId,
		PricePerCore: latestPrice.SpotPrice / float64(meta.CpuCoreCount),
		Price:        fmt.Sprintf("%f", latestPrice.SpotPrice),
		Discount:     10 * latestPrice.SpotPrice / latestPrice.OriginPrice,
		Possibility:  GetPossibility(prices),
	}
	return ip
}

func FindLatestPrice(prices []ecsService.SpotPriceType) ecsService.SpotPriceType {
	var latestPrice ecsService.SpotPriceType

	for _, price := range prices {
		if latestPrice.Timestamp == "" {
			latestPrice = price
		} else {
			latestDate, err := time.Parse(time.RFC3339, fmt.Sprintf("%s", latestPrice.Timestamp))
			if err != nil {
				log.Panicf("Time format is not valid,because of %v", err)
			}

			currentDate, err := time.Parse(time.RFC3339, fmt.Sprintf("%s", price.Timestamp))
			if err != nil {
				log.Panicf("Time format is not valid,because of %v", err)
			}

			if latestDate.Before(currentDate) {
				latestPrice = price
			}
		}
	}

	return latestPrice
}

func GetPossibility(prices []ecsService.SpotPriceType) float64 {
	var variance float64 = 0
	var sigma float64 = 0

	for _, price := range prices {
		variance += math.Pow((price.SpotPrice - 0.1*price.OriginPrice), 2)
	}

	sigma = math.Sqrt(variance / float64(len(prices)))

	return sigma
}
