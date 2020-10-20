package pkg

import (
	"encoding/json"
	"fmt"
	logger "github.com/Sirupsen/logrus"
	"github.com/olekukonko/tablewriter"
	"reflect"
	"strconv"
	"strings"
)

const (
	MDSPACE  = "&#160;"
	SPACENUM = 4
)

var (
	TITLEMAP = map[string]string{
		"InstanceTypeId": "机型",
		"ZoneId":         "可用区",
		"PricePerCore":   "核时价格",
	}
	PREFIXCHARTER = []string{"-", "_", "+", ""}
)

type Formator interface {
	SetOriginData(*[]interface{})
	SetHaveHeader(bool)
	Format() (string, error)
}

type PriceShow struct {
	InstanceTypeId string  `json:"InstanceTypeId" xml:"InstanceTypeId"`
	ZoneId         string  `json:"zone_id"`
	PricePerCore   float64 `json:"price_per_core"`
}

type MarkDownFormator struct {
	Advisor     *Advisor
	OriginData  *[]interface{}
	ToShow      *[]interface{} // e.g. *[]PriceShow
	ShowStruct  interface{}    // e.g. PriceShow
	TableHeader *[]string
	ToShowData  [][]string
	FiledMaxLen map[string]int
	HaveHeader  bool // default true
}

type FormatorConfig struct {
	Name string `json:"name"`
	MarkDownFormator
}

func NewFormator(config *FormatorConfig) Formator {
	if config == nil {
		logger.Error("empty filter config")
		return nil
	}
	switch config.Name {
	case MarkdownType:
		return &MarkDownFormator{
			Advisor:     config.Advisor,
			OriginData:  config.OriginData,
			ToShow:      &[]interface{}{},
			ShowStruct:  config.ShowStruct,
			TableHeader: nil,
			ToShowData:  nil,
			FiledMaxLen: nil,
			HaveHeader:  config.HaveHeader,
		}
	}
	return nil
}

func covertStructsByJson(a interface{}, b interface{}) (interface{}, error) {
	ShowStructType := reflect.TypeOf(b)
	ShowStructPtr := ShowStructType.Elem()
	newItem := reflect.New(ShowStructPtr)

	aJsonBytes, err := json.Marshal(a)
	if err != nil {
		logger.Errorf("covert structs by json: json encode %v err:%v")
		return nil, err
	}
	// reflected pointer
	newP := newItem.Interface()
	err = json.Unmarshal(aJsonBytes, newP)
	if err != nil {
		logger.Errorf(fmt.Sprintf("covertStructsByJson decode:%s error:%v", string(aJsonBytes), err))
		return nil, err
	}
	logger.Debug(fmt.Sprintf("covertStructsByJson decode:%s result:%++v", string(aJsonBytes), newP))

	return newP, nil
}

func (mdf *MarkDownFormator) SetOriginData(originData *[]interface{}) {
	if originData == nil || len(*originData) == 0 {
		return
	}
	if mdf.OriginData == nil {
		mdf.OriginData = &[]interface{}{}
	}
	*mdf.OriginData = *originData
	logger.Debugf("mdf.OriginData %++v\n\n", *mdf.OriginData)
}

func (mdf *MarkDownFormator) SetHaveHeader(have bool) {
	mdf.HaveHeader = have
}

// change to map && get field to show
func (mdf *MarkDownFormator) getToShow() error {
	mdf.ToShow = &[]interface{}{}
	for _, item := range *mdf.OriginData {
		newItem, err := covertStructsByJson(item, mdf.ShowStruct)
		if err != nil {
			logger.Errorf(fmt.Sprintf("error:%v", err))
			continue
		}
		*mdf.ToShow = append(*mdf.ToShow, newItem)
	}
	return nil
}

func (mdf *MarkDownFormator) Format() (string, error) {
	// get the field needed
	if err := mdf.getToShow(); err != nil {
		return "", err
	}
	logger.Debugf("mdf.getToShow is %++v", mdf.ToShow)
	// make to show data cell clean, clear prefix or others
	for _, v := range *mdf.ToShow {
		cleanShowItem(v, mdf.Advisor)
	}
	logger.Debugf("After clean is %++v", *mdf.ToShow)

	// get the max length map of every column
	err := mdf.getMaxLength()
	if err != nil {
		return "", err
	}
	logger.Debugf("mdf.FiledMaxLen is %++v", mdf.FiledMaxLen)

	if len(*mdf.ToShow) > 0 {
		mdf.SetTableHeader((*mdf.ToShow)[0])
	} else {
		return "", fmt.Errorf("nothing to format")
	}
	logger.Debugf("mdf.TableHeader is %++v", *mdf.TableHeader)

	//add space: need to fix parse
	err = mdf.PreFormat()
	//if err != nil {
	//	return "", err
	//}
	logger.Debugf("After PreFormat is %++v", *mdf.ToShow)

	mdf.ToShowData = [][]string{}
	for _, item := range *mdf.ToShow {
		mdf.ToShowData = append(mdf.ToShowData, toStrings(item))
	}

	mdStr, err := doFormat(mdf.TableHeader, mdf.ToShowData)
	if err != nil {
		return "", err
	}

	return mdStr, nil
}

func cleanShowItem(showItem interface{}, advisor *Advisor) {
	switch item := showItem.(type) {
	case *PriceShow:
		item.InstanceTypeId = strings.Replace(item.InstanceTypeId, "ecs.", "", 1)
		regions := strings.Split(advisor.Region, ",")
		for _, region := range regions {
			if strings.Contains(item.ZoneId, region) {
				item.ZoneId = removeZoneIdPrefix(item.ZoneId, region)
			}
		}
		item.PricePerCore, _ = strconv.ParseFloat(fmt.Sprintf("%.5f", item.PricePerCore), 64)
	case PriceShow:
		item.InstanceTypeId = strings.Replace(item.InstanceTypeId, "ecs.", "", 1)
		regions := strings.Split(advisor.Region, ",")
		for _, region := range regions {
			if strings.Contains(item.ZoneId, region) {
				item.ZoneId = removeZoneIdPrefix(item.ZoneId, region)
			}
		}
		item.PricePerCore, _ = strconv.ParseFloat(fmt.Sprintf("%.5f", item.PricePerCore), 64)

	default:
		logger.Errorf("unknown type of show item")
	}
}

func removeZoneIdPrefix(zoneId string, region string) string {
	for _, charter := range PREFIXCHARTER {
		if strings.Contains(zoneId, region+charter) {
			zoneId = strings.Replace(zoneId, region+charter, "", -1)
			return zoneId
		}
	}
	return zoneId
}

func (mdf *MarkDownFormator) getMaxLength() error {
	mdf.FiledMaxLen = map[string]int{}
	for _, item := range *mdf.ToShow {
		mapPtr, err := TransToMap(item)
		if err != nil {
			logger.Errorf("trans to map error:", err)
		}
		for key, val := range *mapPtr {
			valLen := 0
			switch valType := val.(type) {
			case string:
				valLen = len(valType)
			case float64:
				valLen = len(Decimal(valType))
			default:
				logger.Errorf("unknown toshow field type")
				continue
			}
			if _, ok := mdf.FiledMaxLen[key]; !ok {
				mdf.FiledMaxLen[key] = len((TITLEMAP)[key])
			}
			if mdf.FiledMaxLen[key] < valLen {
				mdf.FiledMaxLen[key] = valLen
			}
		}
	}
	return nil
}

func (mdf *MarkDownFormator) SetTableHeader(s interface{}) {
	if !mdf.HaveHeader {
		mdf.TableHeader = &[]string{}
		return
	}
	v := reflect.Indirect(reflect.ValueOf(s))
	var header []string
	for i := 0; i < v.NumField(); i++ {
		fieldName := v.Type().Field(i).Name
		if title, ok := TITLEMAP[fieldName]; ok {
			title = addSpace(title, mdf.FiledMaxLen[fieldName])
			header = append(header, title)
		} else {
			header = append(header, "未命名")
		}
	}
	mdf.TableHeader = &header
}

func (mdf *MarkDownFormator) PreFormat() error {
	for k, v := range *mdf.ToShow {
		maps, err := TransToMap(v)
		if err != nil {
			logger.Errorf("trans to map err:", err)
		}
		for key, val := range *maps {
			(*maps)[key] = addSpace(fmt.Sprintf("%v", val), mdf.FiledMaxLen[key])
		}
		// change map to struct by json
		toStructJson, err := json.Marshal(*maps)
		if err != nil {
			return err
		}
		err = json.Unmarshal(toStructJson, (*mdf.ToShow)[k])
		if err != nil {
			return err
		}
	}
	return nil
}

func doFormat(headers *[]string, toShowData [][]string) (string, error) {
	if len(toShowData) < 1 {
		return "", fmt.Errorf("empty to show data, nothing to show")
	}
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	if headers != nil {
		table.SetHeader(*headers)
	}
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetNoWhiteSpace(false)
	table.AppendBulk(toShowData) // Add Bulk Data
	table.Render()
	return tableString.String(), nil
}

func addSpace(originStr string, maxLen int) string {
	newStr := originStr
	toAddSpaceNum := SPACENUM + ((maxLen - len(originStr)) / 2)
	for i := 0; i < toAddSpaceNum; i++ {
		newStr = newStr + MDSPACE
	}
	return newStr
}

func toStrings(x interface{}) []string {
	v := reflect.Indirect(reflect.ValueOf(x))
	var r []string
	for i := 0; i < v.NumField(); i++ {
		switch v.Field(i).Kind() {
		case reflect.Int64:
			r = append(r, strconv.FormatInt(v.Field(i).Int(), 10))
		case reflect.Float64:
			r = append(r, strconv.FormatFloat(v.Field(i).Float(), 'f', -1, 64))
		case reflect.String:
			r = append(r, v.Field(i).String())
		default:
			r = append(r, v.Field(i).String())
		}
	}
	return r
}
