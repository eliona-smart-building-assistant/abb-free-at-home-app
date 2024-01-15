//  This file is part of the eliona project.
//  Copyright © 2022 LEICOM iTEC AG. All Rights Reserved.
//  ______ _ _
// |  ____| (_)
// | |__  | |_  ___  _ __   __ _
// |  __| | | |/ _ \| '_ \ / _` |
// | |____| | | (_) | | | | (_| |
// |______|_|_|\___/|_| |_|\__,_|
//
//  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING
//  BUT NOT LIMITED  TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
//  NON INFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
//  DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package broker

import (
	"abb-free-at-home/abb"
	"abb-free-at-home/abbgraphql"
	"abb-free-at-home/apiserver"
	"abb-free-at-home/appdb"
	"abb-free-at-home/conf"
	"abb-free-at-home/model"
	"errors"
	"fmt"
	"strconv"
	"strings"

	elionaapi "github.com/eliona-smart-building-assistant/go-eliona-api-client/v2"
	"github.com/eliona-smart-building-assistant/go-utils/log"
)

const (
	function_switch                = "switch"
	function_status                = "status" // For simple abb outputs
	function_dimmer                = "dimmer"
	function_measured_temperature  = "measured_temperature"
	function_set_temperature       = "set_temperature"
	function_heating_flow          = "heating_flow"
	function_actuator_heating_flow = "actuator_heating_flow"
	function_heating_active        = "heating_active"
	function_heating_value         = "heating_value"
	function_status_indication     = "status_indication"
	function_presence              = "presence"
	function_window_door           = "window_door"
	function_hsv                   = "hsv"
	function_hsv_hue               = "hsv_hue"
	function_hsv_saturation        = "hsv_saturation"
	function_hsv_value             = "hsv_value"
	function_color_mode            = "color_mode"
	function_color_temperature     = "color_temperature"
	function_set_scene             = "set_scene"
)

const SET_TEMP_TWICE = function_set_temperature
const SET_SCENE_RETURN_TO_ZERO = function_set_scene

var Functions = []string{
	function_status,
	// Note: Depends on order.
	function_switch,
	function_dimmer,
	function_measured_temperature,
	function_set_temperature,
	function_heating_flow,
	function_actuator_heating_flow,
	function_heating_active,
	function_heating_value,
	function_status_indication,
	function_presence,
	function_window_door,
	function_hsv,
	function_hsv_hue,
	function_hsv_saturation,
	function_hsv_value,
	function_color_mode,
	function_color_temperature,
	function_set_scene,
}

func getAPI(config *apiserver.Configuration) (*abb.Api, error) {
	var api *abb.Api
	switch config.AbbConnectionType {
	case conf.ABB_LOCAL:
		if config.ApiUsername == nil || config.ApiPassword == nil || config.ApiUrl == nil || config.RequestTimeout == nil {
			return nil, fmt.Errorf("one or more required config fields (ApiUsername, ApiPassword, ApiUrl, RequestTimeout) are nil")
		}
		api = abb.NewLocalApi(*config.ApiUsername, *config.ApiPassword, *config.ApiUrl, int(*config.RequestTimeout))
	case conf.ABB_MYBUILDINGS:
		if config.ClientID == nil || config.ClientSecret == nil || config.RequestTimeout == nil {
			return nil, fmt.Errorf("one or more required config fields (ClientID, ClientSecret, RequestTimeout) are nil")
		}
		api = abb.NewMyBuildingsApi(*config)
	case conf.ABB_PROSERVICE:
		if config.ApiKey == nil || config.OrgUUID == nil {
			return nil, fmt.Errorf("api key or org-uuid is missing in config")
		}
		api = abb.NewProServiceApi(*config)
	}
	if err := api.Authorize(); err != nil {
		if _, err := conf.InvalidateAuthorization(*config); err != nil {
			return nil, fmt.Errorf("invalidating authorization: %v", err)
		}
		return nil, fmt.Errorf("authorizing: %v", err)
	}
	if api.Auth.OauthToken != nil {
		if _, err := conf.PersistAuthorization(config, *api.Auth.OauthToken); err != nil {
			return nil, fmt.Errorf("persisting authorization: %v", err)
		}
	}
	return api, nil
}

func GetLocations(config *apiserver.Configuration) ([]model.Floor, error) {
	api, err := getAPI(config)
	if err != nil {
		return nil, fmt.Errorf("getting API instance: %v", err)
	}
	abbLocations, err := api.GetLocations()
	if err != nil && strings.Contains(err.Error(), "UNAUTHENTICATED") {
		if _, err := conf.InvalidateAuthorization(*config); err != nil {
			return nil, fmt.Errorf("invalidating authorization: %v", err)
		}
		return nil, errors.New("authorization invalidated")
	} else if err != nil {
		return nil, fmt.Errorf("getting locations: %v", err)
	}
	var floors []model.Floor
	for _, system := range abbLocations.ISystemFH {
		for _, floor := range system.Locations {
			f := model.Floor{
				Id:    floor.DtId,
				Name:  floor.Label,
				Level: floor.Level,
			}
			for _, room := range floor.Sublocations {
				r := model.Room{
					Id:   room.DtId,
					Name: room.Label,
				}
				f.Rooms = append(f.Rooms, r)
			}
			floors = append(floors, f)
		}
	}
	return floors, nil
}

// GetSystems gets systems according to the configuration passed.
// The systems are then converted to a model.System type and the datapoints are mapped here.
func GetSystems(config *apiserver.Configuration) ([]model.System, error) {
	api, err := getAPI(config)
	if err != nil {
		return nil, fmt.Errorf("getting API instance: %v", err)
	}
	abbConfiguration, err := api.GetConfiguration()
	if err != nil && strings.Contains(err.Error(), "UNAUTHENTICATED") {
		if _, err := conf.InvalidateAuthorization(*config); err != nil {
			return nil, fmt.Errorf("invalidating authorization: %v", err)
		}
		return nil, errors.New("authorization invalidated")
	} else if err != nil {
		return nil, fmt.Errorf("getting configuration: %v", err)
	}

	var systems []model.System
	for id, system := range abbConfiguration.Systems {
		connectionStatus := int8(0)
		if system.ConnectionOK {
			connectionStatus = 1
		}
		s := model.System{
			ID:               id,
			GAI:              id,
			Name:             system.SysApName,
			ConnectionStatus: connectionStatus,
		}
		if adheres, err := s.AdheresToFilter(config.AssetFilter); err != nil {
			return nil, fmt.Errorf("determining whether system adheres to a filter: %v", err)
		} else if !adheres {
			continue
		}
		for id, device := range system.Devices {
			d := model.Device{
				ID:           id,
				GAI:          s.GAI + "_" + id,
				Name:         device.DisplayName.(string),
				Location:     device.Location,
				Battery:      device.Battery,
				Connectivity: device.Connectivity,
			}
			for id, channel := range device.Channels {
				if channel.FunctionId == "" {
					log.Debug("broker", "skipped channel %v with empty functionID", channel.DisplayName)
					continue
				}
				var c model.Asset
				fid, err := strconv.ParseInt(channel.FunctionId, 16, 0)
				if err != nil {
					log.Error("broker", "parsing functionID %s: %v", channel.FunctionId, err)
					continue
				}
				assetBase := model.AssetBase{
					IDBase:   id,
					GAIBase:  d.GAI + "_" + id,
					NameBase: channel.DisplayName.(string),
				}
				switch fid {
				case model.FID_SWITCH_ACTUATOR:
					// Used for ABB -> Eliona
					outputs := make(map[string]model.Datapoint)
					for datapoint, output := range channel.Outputs {
						if output.PairingId == model.PID_ON_OFF_INFO_GET {
							outputs[function_switch] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "switch",
									},
								},
							}
						}
					}
					assetBase.OutputsBase = outputs
					// Used for Eliona -> ABB
					inputs := make(map[string]string)
					for datapoint, input := range channel.Inputs {
						if input.PairingId == model.PID_SWITCH_ON_OFF_SET {
							inputs[function_switch] = datapoint
						}
					}
					assetBase.InputsBase = inputs

					// Used for current values in Eliona one-time update
					switchState := parseInt8(channel.FindOutputValueByPairingID(model.PID_ON_OFF_INFO_GET))
					c = model.Switch{
						AssetBase: assetBase,
						Switch:    switchState,
					}
				case model.FID_DIMMING_ACTUATOR:
					outputs := make(map[string]model.Datapoint)
					for datapoint, output := range channel.Outputs {
						switch output.PairingId {
						case model.PID_ON_OFF_INFO_GET:
							outputs[function_switch] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "switch",
									},
								},
							}
						case model.PID_ACTUAL_DIM_VALUE_0_100_GET:
							outputs[function_dimmer] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "dimmer",
									},
								},
							}
						}
					}
					assetBase.OutputsBase = outputs

					inputs := make(map[string]string)
					for datapoint, input := range channel.Inputs {
						switch input.PairingId {
						case model.PID_SWITCH_ON_OFF_SET:
							inputs[function_switch] = datapoint
						case model.PID_ABSOLUTE_VALUE_0_100_SET:
							inputs[function_dimmer] = datapoint
						}
					}
					assetBase.InputsBase = inputs

					switchState := parseInt8(channel.FindOutputValueByPairingID(model.PID_ON_OFF_INFO_GET))
					dimmerState := parseInt8(channel.FindOutputValueByPairingID(model.PID_ACTUAL_DIM_VALUE_0_100_GET))
					c = model.Dimmer{
						AssetBase: assetBase,
						Switch:    switchState,
						Dimmer:    dimmerState,
					}
				case model.FID_HUE_ACTUATOR:
					outputs := make(map[string]model.Datapoint)
					for datapoint, output := range channel.Outputs {
						switch output.PairingId {
						case model.PID_ON_OFF_INFO_GET:
							outputs[function_switch] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "switch",
									},
								},
							}
						case model.PID_ACTUAL_DIM_VALUE_0_100_GET:
							outputs[function_dimmer] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "dimmer",
									},
								},
							}
						case model.PID_HSV_COLOR_GET:
							outputs[function_hsv] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "hsv_state",
									},
								},
							}
						case model.PID_COLOR_MODE_GET:
							outputs[function_color_mode] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "color_mode_state",
									},
								},
							}
						case model.PID_COLOR_TEMPERATURE_GET:
							outputs[function_color_temperature] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "color_temperature",
									},
								},
							}
						}
					}
					assetBase.OutputsBase = outputs

					inputs := make(map[string]string)
					for datapoint, input := range channel.Inputs {
						switch input.PairingId {
						case model.PID_SWITCH_ON_OFF_SET:
							inputs[function_switch] = datapoint
						case model.PID_ABSOLUTE_VALUE_0_100_SET:
							inputs[function_dimmer] = datapoint
						case model.PID_HSV_HUE_SET:
							inputs[function_hsv_hue] = datapoint
						case model.PID_HSV_SATURATION_SET:
							inputs[function_hsv_saturation] = datapoint
						case model.PID_HSV_VALUE_SET:
							inputs[function_hsv_value] = datapoint
						case model.PID_COLOR_TEMPERATURE_SET:
							inputs[function_color_temperature] = datapoint
						}
					}
					assetBase.InputsBase = inputs

					switchState := parseInt8(channel.FindOutputValueByPairingID(model.PID_ON_OFF_INFO_GET))
					dimmerState := parseInt8(channel.FindOutputValueByPairingID(model.PID_ACTUAL_DIM_VALUE_0_100_GET))
					// TODO: HSV could be calculated for this to populate the three-channel inputs as well.
					hsvState := channel.FindOutputValueByPairingID(model.PID_HSV_COLOR_GET)
					colorMode := channel.FindOutputValueByPairingID(model.PID_COLOR_MODE_GET)
					colorTemperature := parseInt8(channel.FindOutputValueByPairingID(model.PID_COLOR_TEMPERATURE_GET))
					c = model.HueActuator{
						AssetBase:        assetBase,
						Switch:           switchState,
						Dimmer:           dimmerState,
						HSVState:         hsvState,
						ColorModeState:   colorMode,
						ColorTemperature: colorTemperature,
					}
				case model.FID_ROOM_TEMPERATURE_CONTROLLER_MASTER_WITH_FAN, model.FID_ROOM_TEMPERATURE_CONTROLLER_MASTER_WITHOUT_FAN, model.FID_ROOM_TEMPERATURE_CONTROLLER_SLAVE:
					outputs := make(map[string]model.Datapoint)
					for datapoint, output := range channel.Outputs {
						switch output.PairingId {
						case model.PID_CONTROLLER_ON_OFF_PROTECTED_GET:
							outputs[function_switch] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "switch",
									},
								},
							}
						case model.PID_MEASURED_TEMPERATURE:
							outputs[function_measured_temperature] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "current_temperature",
									},
								},
							}
						case model.PID_SETPOINT_TEMPERATURE_GET:
							outputs[function_set_temperature] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "set_temperature",
									},
								},
							}
						}
					}
					assetBase.OutputsBase = outputs

					inputs := make(map[string]string)
					for datapoint, input := range channel.Inputs {
						switch input.PairingId {
						case model.PID_CONTROLLER_REQ_ON_OFF_SET:
							inputs[function_switch] = datapoint
						case model.PID_ABS_TEMPERATURE_SET:
							inputs[function_set_temperature] = datapoint
						}
					}
					assetBase.InputsBase = inputs

					switchState := parseInt8(channel.FindOutputValueByPairingID(model.PID_CONTROLLER_ON_OFF_PROTECTED_GET))
					currentTemp := parseFloat32(channel.FindOutputValueByPairingID(model.PID_MEASURED_TEMPERATURE))
					setTemp := parseFloat32(channel.FindOutputValueByPairingID(model.PID_SETPOINT_TEMPERATURE_GET))
					c = model.RTC{
						AssetBase:   assetBase,
						Switch:      switchState,
						CurrentTemp: float32(currentTemp),
						SetTemp:     float32(setTemp),
					}
				case model.FID_RADIATOR_THERMOSTAT:
					outputs := make(map[string]model.Datapoint)
					for datapoint, output := range channel.Outputs {
						switch output.PairingId {
						case model.PID_CONTROLLER_ON_OFF_PROTECTED_GET:
							outputs[function_switch] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "switch",
									},
								},
							}
						case model.PID_MEASURED_TEMPERATURE:
							outputs[function_measured_temperature] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "current_temperature",
									},
								},
							}
						case model.PID_SETPOINT_TEMPERATURE_GET:
							outputs[function_set_temperature] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "set_temperature",
									},
								},
							}
						case model.PID_HEATING_MODE_GET:
							outputs[function_status_indication] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "status_indication",
									},
								},
							}
						case model.PID_HEATING_ACTIVE:
							outputs[function_heating_active] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "heating_active",
									},
								},
							}
						case model.PID_HEATING_VALUE:
							outputs[function_heating_value] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "heating_value",
									},
								},
							}
						}
					}
					assetBase.OutputsBase = outputs

					inputs := make(map[string]string)
					for datapoint, input := range channel.Inputs {
						switch input.PairingId {
						case model.PID_CONTROLLER_REQ_ON_OFF_SET:
							inputs[function_switch] = datapoint
						case model.PID_ABS_TEMPERATURE_SET:
							inputs[function_set_temperature] = datapoint
						case model.PID_PRESENCE:
							inputs[function_presence] = datapoint
						case model.PID_AL_WINDOW_DOOR:
							inputs[function_window_door] = datapoint
						}
					}
					assetBase.InputsBase = inputs

					switchState := parseInt8(channel.FindOutputValueByPairingID(model.PID_CONTROLLER_ON_OFF_PROTECTED_GET))
					currentTemp := parseFloat32(channel.FindOutputValueByPairingID(model.PID_MEASURED_TEMPERATURE))
					setTemp := parseFloat32(channel.FindOutputValueByPairingID(model.PID_SETPOINT_TEMPERATURE_GET))
					statusIndication := parseInt8(channel.FindOutputValueByPairingID(model.PID_HEATING_MODE_GET))
					heatingActive := parseInt8(channel.FindOutputValueByPairingID(model.PID_HEATING_ACTIVE))
					heatingValue := parseInt8(channel.FindOutputValueByPairingID(model.PID_HEATING_VALUE))
					c = model.RadiatorThermostat{
						AssetBase:        assetBase,
						Switch:           switchState,
						CurrentTemp:      float32(currentTemp),
						SetTemp:          float32(setTemp),
						StatusIndication: statusIndication,
						HeatingActive:    heatingActive,
						HeatingValue:     heatingValue,
					}
				case model.FID_WINDOW_DOOR_SENSOR:
					outputs := make(map[string]model.Datapoint)
					for datapoint, output := range channel.Outputs {
						if output.PairingId == model.PID_AL_WINDOW_DOOR {
							outputs[function_status] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "position",
									},
								},
							}
						}
					}
					assetBase.OutputsBase = outputs

					position := parseInt8(channel.FindOutputValueByPairingID(model.PID_AL_WINDOW_DOOR))
					c = model.DoorSensor{
						AssetBase: assetBase,
						Position:  position,
					}
				case model.FID_WINDOW_DOOR_POSITION_SENSOR:
					outputs := make(map[string]model.Datapoint)
					for datapoint, output := range channel.Outputs {
						if output.PairingId == model.PID_AL_WINDOW_DOOR_POSITION {
							outputs[function_status] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "position",
									},
								},
							}
						}
					}
					assetBase.OutputsBase = outputs

					position := parseInt8(channel.FindOutputValueByPairingID(model.PID_AL_WINDOW_DOOR_POSITION))
					c = model.WindowSensor{
						AssetBase: assetBase,
						Position:  position,
					}
				case model.FID_MOVEMENT_DETECTOR:
					outputs := make(map[string]model.Datapoint)
					for datapoint, output := range channel.Outputs {
						if output.PairingId == model.PID_MOVEMENT_UNDER_CONSIDERATION_OF_BRIGHTNESS {
							outputs[function_status] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "movement",
									},
								},
							}
						}
					}
					assetBase.OutputsBase = outputs

					movement := parseInt8(channel.FindOutputValueByPairingID(model.PID_MOVEMENT_UNDER_CONSIDERATION_OF_BRIGHTNESS))
					c = model.MovementSensor{
						AssetBase: assetBase,
						Movement:  movement,
					}
				case model.FID_SMOKE_DETECTOR:
					outputs := make(map[string]model.Datapoint)
					for datapoint, output := range channel.Outputs {
						if output.PairingId == model.PID_FIRE_ALARM_ACTIVE {
							outputs[function_status] = model.Datapoint{

								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "fire",
									},
								},
							}
						}
					}
					assetBase.OutputsBase = outputs

					fire := parseInt8(channel.FindOutputValueByPairingID(model.PID_FIRE_ALARM_ACTIVE))
					c = model.SmokeDetector{
						AssetBase: assetBase,
						Fire:      fire,
					}
				case model.FID_DES_LEVEL_CALL_SENSOR:
					outputs := make(map[string]model.Datapoint)
					for datapoint, output := range channel.Outputs {
						if output.PairingId == model.PID_TIMED_START_STOP {
							outputs[function_status] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "floor_call",
									},
								},
							}
						}
					}
					assetBase.OutputsBase = outputs

					floorCall := parseInt8(channel.FindOutputValueByPairingID(model.PID_TIMED_START_STOP))
					c = model.FloorCallButton{
						AssetBase: assetBase,
						FloorCall: floorCall,
					}
				case model.FID_HEATING_ACTUATOR:
					outputs := make(map[string]model.Datapoint)
					for datapoint, output := range channel.Outputs {
						if output.PairingId == model.PID_AL_INFO_VALUE_HEATING {
							outputs[function_heating_flow] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "info_flow",
									},
								},
							}
						}
						if output.PairingId == model.PID_ACTUATING_VALUE_HEATING {
							outputs[function_actuator_heating_flow] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "actuator_flow",
									},
								},
							}
						}
					}
					assetBase.OutputsBase = outputs

					infoFlow := parseInt8(channel.FindOutputValueByPairingID(model.PID_AL_INFO_VALUE_HEATING))
					actuatorFlow := parseInt8(channel.FindOutputValueByPairingID(model.PID_ACTUATING_VALUE_HEATING))
					c = model.HeatingActuator{
						AssetBase:    assetBase,
						InfoFlow:     infoFlow,
						ActuatorFlow: actuatorFlow,
					}
				case model.FID_SCENE, model.FID_SPECIAL_SCENE_PANIC, model.FID_SPECIAL_SCENE_ALL_OFF, model.FID_SPECIAL_SCENE_ALL_BLINDS_UP, model.FID_SPECIAL_SCENE_ALL_BLINDS_DOWN:
					// Scenes are stateless, therefore we cannot read their state. We can only control them.
					// But we need to simulate this state in Eliona, to allow a "trigger" UX on the attribute.
					// That's why we need to specify the outputs as well.
					outputs := make(map[string]model.Datapoint)
					for datapoint, output := range channel.Outputs {
						if output.PairingId == model.PID_AL_SCENE_CONTROL {
							outputs[function_set_scene] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "set_scene",
									},
								},
							}
						}
					}
					assetBase.OutputsBase = outputs

					// We can control scenes only via output datapoint. Don't ask me why, they don't have
					// any input ones.
					inputs := make(map[string]string)
					inputs[function_set_scene] = "odp0000" // Yes, this is really a setable output.
					assetBase.InputsBase = inputs

					switchState := int8(0) // Scenes are stateless. It's always zero.
					c = model.Scene{
						AssetBase: assetBase,
						Switch:    switchState,
					}
				default:
					continue // Don't create any asset if user cannot work with it.
					// c = model.Channel{
					// 	AssetBase: assetBase,
					// }
				}
				d.Channels = append(d.Channels, c)
			}
			s.Devices = append(s.Devices, d)
		}
		systems = append(systems, s)
	}
	return systems, nil
}

func parseInt8(str string) int8 {
	if str == "" {
		return int8(0)
	}
	i, err := strconv.ParseInt(str, 10, 8)
	if err != nil {
		log.Error("broker", "parsing value '%s': %v", str, err)
	}
	return int8(i)
}

func parseFloat32(str string) float32 {
	if str == "" {
		return float32(0)
	}
	f, err := strconv.ParseFloat(str, 16)
	if err != nil {
		log.Error("broker", "parsing value '%s': %v", str, err)
	}
	return float32(f)
}

func ListenForDataChanges(config *apiserver.Configuration, datapoints []appdb.Datapoint, ch chan<- abbgraphql.DataPoint) error {
	api, err := getAPI(config)
	if err != nil {
		return fmt.Errorf("getting API instance: %v", err)
	}
	err = api.ListenGraphQLSubscriptions(datapoints, ch)
	if err != nil && strings.Contains(err.Error(), "JsonWebTokenError") {
		if _, err := conf.InvalidateAuthorization(*config); err != nil {
			return fmt.Errorf("invalidating authorization: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("listen for graphQL subscriptions: %v", err)
	}
	return nil
}

func ListenForSystemStatusChanges(config *apiserver.Configuration, dtIDs []string, ch chan<- abbgraphql.ConnectionStatus) error {
	api, err := getAPI(config)
	if err != nil {
		return fmt.Errorf("getting API instance: %v", err)
	}
	err = api.ListenGraphQLSystemStatus(dtIDs, ch)
	if err != nil && strings.Contains(err.Error(), "JsonWebTokenError") {
		if _, err := conf.InvalidateAuthorization(*config); err != nil {
			return fmt.Errorf("invalidating authorization: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("listen for system status changes: %v", err)
	}
	return nil
}

func SetInput(config *apiserver.Configuration, input appdb.Datapoint, value float64) error {
	api, err := getAPI(config)
	if err != nil {
		return fmt.Errorf("getting API instance: %v", err)
	}
	return api.WriteDatapoint(input.SystemID, input.DeviceID, input.ChannelID, input.Datapoint, value)
}
