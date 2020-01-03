package main

import (
	"fmt"
	ecsService "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/fatih/color"
	"sort"
	"strings"
	"time"
)

const (
	TimeLayout = "2006-01-02T15:04:05Z"
)

type MetaStore struct {
	*ecsService.Client
	InstanceFamilyCache map[string]ecsService.InstanceType
}

// Initialize the instance type
func (ms *MetaStore) Initialize(region string) {
	req := ecsService.CreateDescribeInstanceTypesRequest()
	req.RegionId = region
	resp, err := ms.DescribeInstanceTypes(req)
	if err != nil {
		panic(fmt.Sprintf("Failed to DescribeInstanceTypes,because of %v", err))
	}
	instanceTypes := resp.InstanceTypes.InstanceType

	for _, instanceType := range instanceTypes {
		ms.InstanceFamilyCache[instanceType.InstanceTypeId] = instanceType
	}

	d_req := ecsService.CreateDescribeAvailableResourceRequest()
	d_req.RegionId = region
	d_req.DestinationResource = "InstanceType"
	d_req.InstanceChargeType = "PostPaid"
	d_req.SpotStrategy = "SpotWithPriceLimit"
	d_resp, err := ms.DescribeAvailableResource(d_req)
	if err != nil {
		panic(fmt.Sprintf("Failed to get available resource,because of %v", err))
	}

	zoneStocks := d_resp.AvailableZones.AvailableZone

	for instanceTypeId := range ms.InstanceFamilyCache {
		found := 0
		for _, zoneStock := range zoneStocks {
			for _, resource := range zoneStock.AvailableResources.AvailableResource[0].SupportedResources.SupportedResource {
				if resource.Value == instanceTypeId {
					found = 1
					break
				}
			}
			if found == 1 {
				break
			}
		}
		if found == 0 {
			delete(ms.InstanceFamilyCache, instanceTypeId)
		}
	}

	fmt.Printf("Initialize cache ready with %d kinds of instanceTypes\n", len(instanceTypes))
}

// Get the instanceType with in the range.
func (ms *MetaStore) FilterInstances(cpu, memory, maxCpu, maxMemory int, family string) (instanceTypes []string) {
	instanceTypes = make([]string, 0)

	instancesFamily := strings.Split(family, ",")

	for key, instanceType := range ms.InstanceFamilyCache {
		if instanceType.CpuCoreCount >= cpu && instanceType.CpuCoreCount <= maxCpu &&
			instanceType.MemorySize >= float64(memory) && instanceType.MemorySize <= float64(maxMemory) {

			for _, instanceFamily := range instancesFamily {
				if strings.Contains(key, instanceFamily) {
					instanceTypes = append(instanceTypes, key)
					break
				}
			}

		}
	}

	fmt.Printf("Filter %d of %d kinds of instanceTypes.\n", len(instanceTypes), len(ms.InstanceFamilyCache))

	return instanceTypes
}

// Fetch spot price history
func (ms *MetaStore) FetchSpotPrices(instanceTypes []string, resolution int) (historyPrices map[string][]ecsService.SpotPriceType) {

	historyPrices = make(map[string][]ecsService.SpotPriceType)

	for _, instanceType := range instanceTypes {
		req := ecsService.CreateDescribeSpotPriceHistoryRequest()
		req.NetworkType = "vpc"
		req.InstanceType = instanceType
		req.IoOptimized = "optimized"
		resp, err := ms.DescribeSpotPriceHistory(req)

		resolutionDuration := time.Duration(resolution * -1*24) * time.Hour
		req.StartTime = time.Now().Add(resolutionDuration).Format(TimeLayout)
		if err != nil {
			continue
		}

		historyPrices[instanceType] = resp.SpotPrices.SpotPriceType
	}

	fmt.Printf("Fetch %d kinds of InstanceTypes prices successfully.\n", len(instanceTypes))

	return historyPrices
}

// Print spot history sort and rank
func (ms *MetaStore) SpotPricesAnalysis(historyPrices map[string][]ecsService.SpotPriceType) SortedInstancePrices {
	sp := make(SortedInstancePrices, 0)
	for instanceTypeId, prices := range historyPrices {
		var meta ecsService.InstanceType
		if m, ok := ms.InstanceFamilyCache[instanceTypeId]; !ok {
			continue
		} else {
			meta = m
		}

		priceAZMap := make(map[string][]ecsService.SpotPriceType)
		for _, price := range prices {
			if priceAZMap[price.ZoneId] == nil {
				priceAZMap[price.ZoneId] = make([]ecsService.SpotPriceType, 0)
			}
			priceAZMap[price.ZoneId] = append(priceAZMap[price.ZoneId], price)
		}

		for zoneId, price := range priceAZMap {
			ip := CreateInstancePrice(meta, zoneId, price)
			sp = append(sp, ip)
		}
	}

	fmt.Printf("Successfully compare %d kinds of instanceTypes\n", len(sp))
	return sp
}

func (ms *MetaStore) PrintPriceRank(prices SortedInstancePrices, cutoff int, limit int) {
	sort.Sort(prices)

	color.Green("%30s %20s %15s %15s %15s\n", "InstanceTypeId", "ZoneId", "Price(Core)", "Discount", "ratio")

	for index, price := range prices {
		if index >= limit {
			break
		}
		if price.Discount <= float64(cutoff) {
			color.Green("%30s %20s %15.4f %15.1f %15.1f\n", price.InstanceTypeId, price.ZoneId, price.PricePerCore, price.Discount, price.Possibility)
		} else {
			color.Blue("%30s %20s %15.4f %15.1f %15.1f\n", price.InstanceTypeId, price.ZoneId, price.PricePerCore, price.Discount, price.Possibility)
		}
	}
}

func NewMetaStore(client *ecsService.Client) *MetaStore {
	return &MetaStore{
		Client:              client,
		InstanceFamilyCache: make(map[string]ecsService.InstanceType),
	}
}
