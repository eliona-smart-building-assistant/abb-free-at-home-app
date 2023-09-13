package model

import (
	"abb-free-at-home/apiserver"
	"fmt"
	"reflect"

	"github.com/eliona-smart-building-assistant/go-utils/common"
)

type System struct {
	ID      string `eliona:"system_id,filterable"`
	GAI     string `eliona:"system_id,filterable"`
	Name    string `eliona:"system_name,filterable"`
	Devices []Device
}

type Device struct {
	ID       string `eliona:"device_id,filterable"`
	GAI      string
	Name     string `eliona:"device_name,filterable"`
	Channels []Asset
}

type Asset interface {
	AssetType() string
	GAI() string
	Name() string
	Id() string
	Inputs() map[string]string // map[function]datapoint
}

type AssetBase struct {
	IDBase     string `eliona:"channel_id,filterable"`
	GAIBase    string
	NameBase   string `eliona:"channel_name,filterable"`
	InputsBase map[string]string
}

func (a AssetBase) Name() string {
	return a.NameBase
}

func (a AssetBase) Id() string {
	return a.IDBase
}

func (a AssetBase) Inputs() map[string]string {
	return a.InputsBase
}

type Channel struct {
	AssetBase
}

func (c Channel) AssetType() string {
	return "abb_free_at_home_channel"
}

func (c Channel) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.GAIBase)
}

type Switch struct {
	AssetBase
	SwitchState int8 `eliona:"switch_state" subtype:"input"`
	Switch      int8 `eliona:"switch" subtype:"output"`
}

func (c Switch) AssetType() string {
	return "abb_free_at_home_switch_sensor"
}

func (c Switch) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.GAIBase)
}

type Dimmer struct {
	AssetBase
	SwitchState int8 `eliona:"switch_state" subtype:"input"`
	DimmerState int8 `eliona:"dimmer_state" subtype:"input"`
	Switch      int8 `eliona:"switch" subtype:"output"`
	Dimmer      int8 `eliona:"dimmer" subtype:"output"`
}

func (c Dimmer) AssetType() string {
	return "abb_free_at_home_dimmer_sensor"
}

func (c Dimmer) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.GAIBase)
}

func (tag *System) AdheresToFilter(filter [][]apiserver.FilterRule) (bool, error) {
	f := apiFilterToCommonFilter(filter)
	fp, err := structToMap(tag)
	if err != nil {
		return false, fmt.Errorf("converting strict to map: %v", err)
	}
	adheres, err := common.Filter(f, fp)
	if err != nil {
		return false, err
	}
	return adheres, nil
}

func structToMap(input interface{}) (map[string]string, error) {
	if input == nil {
		return nil, fmt.Errorf("input is nil")
	}

	inputValue := reflect.ValueOf(input)
	inputType := reflect.TypeOf(input)

	if inputValue.Kind() == reflect.Ptr {
		inputValue = inputValue.Elem()
		inputType = inputType.Elem()
	}

	if inputValue.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input is not a struct")
	}

	output := make(map[string]string)
	for i := 0; i < inputValue.NumField(); i++ {
		fieldValue := inputValue.Field(i)
		fieldType := inputType.Field(i)
		output[fieldType.Name] = fieldValue.String()
	}

	return output, nil
}

func apiFilterToCommonFilter(input [][]apiserver.FilterRule) [][]common.FilterRule {
	result := make([][]common.FilterRule, len(input))
	for i := 0; i < len(input); i++ {
		result[i] = make([]common.FilterRule, len(input[i]))
		for j := 0; j < len(input[i]); j++ {
			result[i][j] = common.FilterRule{
				Parameter: input[i][j].Parameter,
				Regex:     input[i][j].Regex,
			}
		}
	}
	return result
}
