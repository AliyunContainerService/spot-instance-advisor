package main

import (
	"flag"
	ecsService "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"fmt"
)

var (
	accessKeyId     = flag.String("accessKeyId", "", "Your accessKeyId of cloud account")
	accessKeySecret = flag.String("accessKeySecret", "", "Your accessKeySecret of cloud account")
	region          = flag.String("region", "cn-hangzhou", "The region of spot instances")
	cpu             = flag.Int("mincpu", 1, "Min cores of spot instances")
	memory          = flag.Int("minmem", 2, "Min memory of spot instances")
	maxCpu          = flag.Int("maxcpu", 32, "Max cores of spot instances ")
	maxMemory       = flag.Int("maxmem", 64, "Max memory of spot instances")
	family          = flag.String("family", "", "The spot instance family you want (e.g. ecs.n1,ecs.n2)")
	cutoff          = flag.Int("cutoff", 2, "Discount of the spot instance prices")
	limit           = flag.Int("limit", 20, "Limit of the spot instances")
	resolution      = flag.Int("resolution", 7, "The window of price history analysis")
)

func main() {
	flag.Parse()

	client, err := ecsService.NewClientWithAccessKey(*region, *accessKeyId, *accessKeySecret)
	if err != nil {
		panic(fmt.Sprintf("Failed to create ecs client,because of %v", err))
	}

	metastore := NewMetaStore(client)

	metastore.Initialize(*region)

	instanceTypes := metastore.FilterInstances(*cpu, *memory, *maxCpu, *maxMemory, *family)

	historyPrices := metastore.FetchSpotPrices(instanceTypes, *resolution)

	sortedInstancePrices := metastore.SpotPricesAnalysis(historyPrices)

	metastore.PrintPriceRank(sortedInstancePrices, *cutoff, *limit)
}
