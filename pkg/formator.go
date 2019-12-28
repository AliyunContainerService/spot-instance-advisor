package pkg

import (
	"encoding/json"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"reflect"
	"strconv"
	"strings"
)

const (
	MDSPACE  = "&#160;"
	SPACENUM = 4
)

type Formator interface {
	Format() (string, error)
}

type MarkDownFormator struct {
	OriginData  *[]AdvisorResponse
	TableHeader *[]string
	ToShowData  [][]string
	FiledMaxLen map[string]int
}

var (
	TITLEMAP = map[string]string{
		"InstanceTypeId": "机型",
		"ZoneId":         "可用区",
		"PricePerCore":   "核时价格",
	}
)

func (md *MarkDownFormator) getMaxLength() error {
	md.FiledMaxLen = map[string]int{}
	var toMap []map[string]string
	jsonStr, err := json.Marshal(*(md.OriginData))
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonStr, &toMap)
	if err != nil {
		return err
	}

	for _, v := range toMap {
		for key, val := range v {
			if _, ok := md.FiledMaxLen[key]; !ok {
				md.FiledMaxLen[key] = len((TITLEMAP)[key])
			}
			if md.FiledMaxLen[key] < len(val) {
				md.FiledMaxLen[key] = len(val)
			}
		}
	}
	return nil
}

func (md *MarkDownFormator) PreFormat() error {
	var toMap []map[string]string
	jsonStr, err := json.Marshal(*(md.OriginData))
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonStr, &toMap)
	if err != nil {
		return err
	}
	// add
	for _, v := range toMap {
		for key, val := range v {
			v[key] = addSpace(val, md.FiledMaxLen[key])
		}
	}

	toStructJson, err := json.Marshal(toMap)
	if err != nil {
		return err
	}

	err = json.Unmarshal(toStructJson, md.OriginData)
	if err != nil {
		return err
	}
	return nil
}

func (md *MarkDownFormator) Format() (string, error) {
	if len(*md.OriginData) < 1 {
		return "", fmt.Errorf("empty md.OriginData nothing to show")
	}
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader(*md.TableHeader)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetNoWhiteSpace(false)
	table.AppendBulk(md.ToShowData) // Add Bulk Data
	table.Render()
	return tableString.String(), nil
}

func (md *MarkDownFormator) SetTableHeader(s interface{}) {
	v := reflect.ValueOf(s)
	var header []string
	for i := 0; i < v.NumField(); i++ {
		fieldName := v.Type().Field(i).Name
		if title, ok := TITLEMAP[fieldName]; ok {
			// add to header
			title = addSpace(title, md.FiledMaxLen[fieldName])
			header = append(header, title)
		} else {
			header = append(header, "未命名")
		}
	}
	md.TableHeader = &header
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

func getFormator(formatType string, data *[]AdvisorResponse) (Formator, error) {
	switch formatType {
	case "markdown":
		mdf := &MarkDownFormator{
			OriginData:  data,
			TableHeader: nil,
		}
		err := mdf.getMaxLength()
		if err != nil {
			return nil, err
		}
		if len(*data) > 0 {
			mdf.SetTableHeader((*data)[0])
		} else {
			return nil, fmt.Errorf("nothing to format")
		}
		err = mdf.PreFormat()
		if err != nil {
			return nil, err
		}

		for _, item := range *mdf.OriginData {
			mdf.ToShowData = append(mdf.ToShowData, toStrings(item))
		}
		return mdf, nil
	default:
		return nil, fmt.Errorf("unknown format type")
	}
}

func DoFormat(formatType string, toFormat *[]AdvisorResponse) (string, error) {
	if len(*toFormat) == 0 {
		return "", fmt.Errorf("nothing to format")
	}
	formator, err := getFormator(formatType, toFormat)
	if formator == nil || err != nil {
		return "", err
	}
	return formator.Format()
}
