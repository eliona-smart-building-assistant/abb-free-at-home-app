package model

import (
	"abb-free-at-home/apiserver"
	"fmt"
	"reflect"

	api "github.com/eliona-smart-building-assistant/go-eliona-api-client/v2"
	"github.com/eliona-smart-building-assistant/go-utils/common"
)

type Floor struct {
	Id    string
	Name  string
	Level string `eliona:"level" subtype:"info"`
	Rooms []Room
}

func (f Floor) AssetType() string {
	return "abb_free_at_home_floor"
}

func (f Floor) GAI() string {
	return fmt.Sprintf("%s_%s", f.AssetType(), f.Id)
}

type Room struct {
	Id   string
	Name string
}

func (r Room) AssetType() string {
	return "abb_free_at_home_room"
}

func (r Room) GAI() string {
	return fmt.Sprintf("%s_%s", r.AssetType(), r.Id)
}

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
	Location string
	Channels []Asset
}

type Asset interface {
	AssetType() string
	GAI() string
	Name() string
	Id() string
	Inputs() map[string]string     // map[function]datapoint
	Outputs() map[string]Datapoint // map[function]datapoint
}

type AssetBase struct {
	IDBase      string `eliona:"channel_id,filterable"`
	GAIBase     string
	NameBase    string `eliona:"channel_name,filterable"`
	InputsBase  map[string]string
	OutputsBase map[string]Datapoint
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

func (a AssetBase) Outputs() map[string]Datapoint {
	return a.OutputsBase
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

type RTC struct {
	AssetBase

	SwitchState  int8    `eliona:"switch_state" subtype:"input"`
	CurrentTemp  float32 `eliona:"current_temperature" subtype:"input"`
	SetTempState float32 `eliona:"set_temperature_state" subtype:"input"`
	EcoModeState int8    `eliona:"eco_mode_state" subtype:"input"`

	Switch  int8    `eliona:"switch" subtype:"output"`
	SetTemp float32 `eliona:"set_temperature" subtype:"output"`
	EcoMode int8    `eliona:"eco_mode" subtype:"output"`
}

func (rtc RTC) AssetType() string {
	return "abb_free_at_home_room_temperature_controller"
}

func (rtc RTC) GAI() string {
	return fmt.Sprintf("%s_%s", rtc.AssetType(), rtc.GAIBase)
}

type HeatingActuator struct {
	AssetBase
	InfoFlow     int8 `eliona:"info_flow" subtype:"input"`
	ActuatorFlow int8 `eliona:"actuator_flow" subtype:"input"`
}

func (c HeatingActuator) AssetType() string {
	return "abb_free_at_home_heating_actuator"
}

func (c HeatingActuator) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.GAIBase)
}

type WindowSensor struct {
	AssetBase
	Position int8 `eliona:"position" subtype:"input"`
}

func (c WindowSensor) AssetType() string {
	return "abb_free_at_home_window_sensor"
}

func (c WindowSensor) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.GAIBase)
}

type DoorSensor struct {
	AssetBase
	Position int8 `eliona:"position" subtype:"input"`
}

func (c DoorSensor) AssetType() string {
	return "abb_free_at_home_door_sensor"
}

func (c DoorSensor) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.GAIBase)
}

type MovementSensor struct {
	AssetBase
	Movement int8 `eliona:"movement" subtype:"input"`
}

func (c MovementSensor) AssetType() string {
	return "abb_free_at_home_movement_sensor"
}

func (c MovementSensor) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.GAIBase)
}

//

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

type DatapointMap []struct {
	Subtype       api.DataSubtype
	AttributeName string
}

// Datapoint maps ABB datapoint to multiple attributes in Eliona.
type Datapoint struct {
	Name string
	Map  DatapointMap
}
