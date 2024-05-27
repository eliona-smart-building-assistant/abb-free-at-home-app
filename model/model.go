package model

import (
	"abb-free-at-home/apiserver"
	"fmt"

	api "github.com/eliona-smart-building-assistant/go-eliona-api-client/v2"
	"github.com/eliona-smart-building-assistant/go-eliona/utils"
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
	ID               string `eliona:"system_id,filterable"`
	GAI              string `eliona:"system_id,filterable"`
	Name             string `eliona:"system_name,filterable"`
	ConnectionStatus int8   `eliona:"connection_status" subtype:"status"`
	Devices          []Device
}

func (s System) AssetType() string {
	return "abb_free_at_home_system"
}

type Device struct {
	ID           string `eliona:"device_id,filterable"`
	GAI          string
	Name         string `eliona:"device_name,filterable"`
	Location     string
	Battery      *int64 `eliona:"battery" subtype:"status"`
	Connectivity string `eliona:"connectivity" subtype:"status"`
	Channels     []Asset
}

func (d Device) AssetType() string {
	return "abb_free_at_home_device"
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
	Switch int8 `eliona:"switch" subtype:"output"`
}

func (c Switch) AssetType() string {
	return "abb_free_at_home_switch_sensor"
}

func (c Switch) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.GAIBase)
}

type Dimmer struct {
	AssetBase
	Switch int8 `eliona:"switch" subtype:"output"`
	Dimmer int8 `eliona:"dimmer" subtype:"output"`
}

func (c Dimmer) AssetType() string {
	return "abb_free_at_home_dimmer_sensor"
}

func (c Dimmer) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.GAIBase)
}

type HueActuator struct {
	AssetBase
	HSVState         string `eliona:"hsv_state" subtype:"input"`
	ColorModeState   string `eliona:"color_mode_state" subtype:"input"`
	Switch           int8   `eliona:"switch" subtype:"output"`
	Dimmer           int8   `eliona:"dimmer" subtype:"output"`
	HSVHue           int16  `eliona:"hsv_hue" subtype:"output"`
	HSVSaturation    int8   `eliona:"hsv_saturation" subtype:"output"`
	HSVValue         int8   `eliona:"hsv_value" subtype:"output"`
	ColorTemperature int8   `eliona:"color_temperature" subtype:"output"`
}

func (c HueActuator) AssetType() string {
	return "abb_free_at_home_hue_actuator"
}

func (c HueActuator) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.GAIBase)
}

type RTC struct {
	AssetBase

	CurrentTemp float32 `eliona:"current_temperature" subtype:"input"`

	Switch  int8    `eliona:"switch" subtype:"output"`
	SetTemp float32 `eliona:"set_temperature" subtype:"output"`
}

func (rtc RTC) AssetType() string {
	return "abb_free_at_home_room_temperature_controller"
}

func (rtc RTC) GAI() string {
	return fmt.Sprintf("%s_%s", rtc.AssetType(), rtc.GAIBase)
}

type RadiatorThermostat struct {
	AssetBase

	CurrentTemp      float32 `eliona:"current_temperature" subtype:"input"`
	StatusIndication int8    `eliona:"status_indication" subtype:"input"`
	HeatingActive    int8    `eliona:"heating_active" subtype:"input"`
	HeatingValue     int8    `eliona:"heating_value" subtype:"input"`

	Switch  int8    `eliona:"switch" subtype:"output"`
	SetTemp float32 `eliona:"set_temperature" subtype:"output"`
}

func (rt RadiatorThermostat) AssetType() string {
	return "abb_free_at_home_radiator_thermostat"
}

func (rt RadiatorThermostat) GAI() string {
	return fmt.Sprintf("%s_%s", rt.AssetType(), rt.GAIBase)
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

type SmokeDetector struct {
	AssetBase
	Fire int8 `eliona:"fire" subtype:"input"`
}

func (c SmokeDetector) AssetType() string {
	return "abb_free_at_home_smoke_detector"
}

func (c SmokeDetector) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.GAIBase)
}

type FloorCallButton struct {
	AssetBase
	FloorCall int8 `eliona:"floor_call" subtype:"output"`
}

func (c FloorCallButton) AssetType() string {
	return "abb_free_at_home_floor_call_button"
}

func (c FloorCallButton) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.GAIBase)
}

type MuteButton struct {
	AssetBase
	Mute int8 `eliona:"mute_button" subtype:"output"`
}

func (c MuteButton) AssetType() string {
	return "abb_free_at_home_mute_button"
}

func (c MuteButton) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.GAIBase)
}

type Scene struct {
	AssetBase
	Switch int8 `eliona:"set_scene" subtype:"output"`
}

func (c Scene) AssetType() string {
	return "abb_free_at_home_scene"
}

func (c Scene) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.GAIBase)
}

type Wallbox struct {
	AssetBase
	Switch            int8    `eliona:"switch" subtype:"output"`
	Enable            int8    `eliona:"enable" subtype:"output"`
	InstalledPower    float64 `eliona:"installed_power" subtype:"status"`
	TotalEnergy       float64 `eliona:"total_energy" subtype:"input"`
	StartLastCharging string  `eliona:"start_last_charging" subtype:"input"`
	Status            string  `eliona:"status" subtype:"input"`
}

func (c Wallbox) AssetType() string {
	return "abb_free_at_home_wallbox"
}

func (c Wallbox) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.GAIBase)
}

//

func (sys *System) AdheresToFilter(filter [][]apiserver.FilterRule) (bool, error) {
	f := apiFilterToCommonFilter(filter)
	fp, err := utils.StructToMap(sys)
	if err != nil {
		return false, fmt.Errorf("converting struct to map: %v", err)
	}
	adheres, err := common.Filter(f, fp)
	if err != nil {
		return false, err
	}
	return adheres, nil
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
