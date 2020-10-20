package pkg

import (
	"encoding/json"
	"fmt"
	logger "github.com/Sirupsen/logrus"
)

func Decimal(value float64) string {
	valueStr := fmt.Sprintf("%.5f", value)
	return valueStr
}

func TransMapToJsonBytes(req map[string][]string) ([]byte, error) {
	if len(req) == 0 {
		return []byte{}, fmt.Errorf("input cannot be empty")
	}
	toTrans := make(map[string]string, len(req))

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

// trans struct to Map
func TransToMap(stru interface{}) (*map[string]interface{}, error) {
	var advisorMap map[string]interface{}
	//fmt.Printf("input struct:%++v\n", stru)
	inrec, err := json.Marshal(stru)
	//fmt.Println("after marshal:", string(inrec))
	if err != nil {
		logger.Errorf("json encode err:%v")
		return nil, err
	}
	json.Unmarshal(inrec, &advisorMap)
	//fmt.Println("after decode:", advisorMap)
	return &advisorMap, nil

}

// covert InstancePrice to interface slice
func TransPriceSlice2ISlice(structSlice *[]InstancePrice) *[]interface{} {
	interfaceSlice := make([]interface{}, len(*structSlice))
	for i, d := range *structSlice {
		interfaceSlice[i] = d
	}
	return &interfaceSlice
}
