package pkg

import (
	"fmt"
	"net/url"
)

type Request struct {
	Vals *url.Values
}

func NewRequest(vals *url.Values) *Request {
	newDDReq := &Request{
		Vals: &url.Values{},
	}
	if vals != nil {
		*newDDReq.Vals = *vals
	}
	return newDDReq
}

func TransToJsonBytes(vals *url.Values, cfg *Config, notEmptyField *[]string) ([]byte, error) {
	// check request have mandatory fields or not
	if notEmptyField != nil {
		*notEmptyField = append(*cfg.DefaultNotEmptyField, *notEmptyField...)
	}
	if ok, err := PreCheck(*vals, notEmptyField); err != nil || !ok {
		return nil, err
	}

	setAliyunAKDefaults(vals, cfg)
	jsonBytes, err := TransMapToJsonBytes(*vals)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}

func PreCheck(vals url.Values, notEmptyField *[]string) (bool, error) {
	if notEmptyField != nil {
		for _, field := range *notEmptyField {
			if item, ok := vals[field]; !ok || len(item) == 0 {
				return false, fmt.Errorf("EMPTY " + field)
			}
		}
	}
	return true, nil
}

func setAliyunAKDefaults(vals *url.Values, cfg *Config) {
	if v, ok := (*vals)["access_key_id"]; !ok || len(v) == 0 || len(v[0]) == 0 {
		(*vals)["access_key_id"] = []string{cfg.AccessKeyId}
	}
	if v, ok := (*vals)["access_key_secret"]; !ok || len(v) == 0 || len(v[0]) == 0 {
		(*vals)["access_key_secret"] = []string{cfg.AccessKeySecret}
	}
}
