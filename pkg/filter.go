package pkg

import (
	"fmt"
	logger "github.com/Sirupsen/logrus"
	"strconv"
	"time"
)

const DEFAULTEXPIREDTIME = 24 * 60 * 60

// default filter aims to filter out already alarmed and not changed one
type Filter interface {
	Filt(interface{}) *[]InstancePrice
}

type PriceCoreFilter struct {
	PriceCoreMax float64
}

type CutoffFilter struct {
	CutoffMin float64
}

type FilterData struct {
	Currents *[]InstancePrice
	Showed   *map[string]*ChangedOne
}

type ChangedOne struct {
	Current         *InstancePrice
	Last            *InstancePrice
	UpdateTimeStamp int64
}

func NewFilter(config *FilterConfig) Filter {
	if config == nil {
		return nil
	}
	switch config.JudgedByField {
	case "cutoff":
		// todo
		return nil
	default:
		threshold, err := strconv.ParseFloat(config.Threshold, 64)
		if err != nil {
			logger.Errorf("parse str to float error: ", err.Error())
			return nil
		}
		filter := PriceCoreFilter{PriceCoreMax: threshold}
		return &filter
	}
}

func (priceCoreFilter *PriceCoreFilter) Filt(toFilter interface{}) *[]InstancePrice {
	switch i := toFilter.(type) {
	case *FilterData:
		var filtered []InstancePrice
		for _, item := range *i.Currents {
			if priceCoreFilter.isNewChanged(item, i.Showed) {
				if filtered == nil {
					filtered = []InstancePrice{}
				}
				filtered = append(filtered, item)
			}
		}
		if filtered == nil {
			return nil
		}
		return &filtered
	}
	return nil
}

func (priceCoreFilter *PriceCoreFilter) isNewChanged(toMap InstancePrice, showed *map[string]*ChangedOne) bool {
	if expireShowed(showed, DEFAULTEXPIREDTIME) {
		logger.Warnf("after expired len(*showed) is ", len(*showed))
	}

	isShow := true
	key := fmt.Sprintf("%s#%s", toMap.InstanceTypeId, toMap.ZoneId)
	if chPrice, ok := (*showed)[key]; ok {
		chPrice.Last, chPrice.Current, chPrice.UpdateTimeStamp =
			chPrice.Current, &toMap, time.Now().Unix()
		if (chPrice.Last.PricePerCore >= priceCoreFilter.PriceCoreMax && chPrice.Current.PricePerCore >= priceCoreFilter.PriceCoreMax) ||
			(chPrice.Last.PricePerCore < priceCoreFilter.PriceCoreMax && chPrice.Current.PricePerCore < priceCoreFilter.PriceCoreMax) {
			isShow = false
		}
	} else {
		(*showed)[key] = &ChangedOne{
			Current:         &toMap,
			Last:            &toMap,
			UpdateTimeStamp: time.Now().Unix(),
		}
	}
	return isShow
}

func expireShowed(toExpire *map[string]*ChangedOne, timeLimit int64) bool {
	expired := false
	for k, v := range *toExpire {
		if (time.Now().Unix() - (*v).UpdateTimeStamp) >= timeLimit {
			expired = true
			logger.Warnf("going to delete ", k)
			delete(*toExpire, k)
			logger.Warnf("length of showed is:", len(*toExpire))
		}
	}
	return expired
}
