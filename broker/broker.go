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
	"fmt"
	"strconv"

	elionaapi "github.com/eliona-smart-building-assistant/go-eliona-api-client/v2"
	"github.com/eliona-smart-building-assistant/go-utils/log"
)

const (
	function_switch               = "switch"
	function_dimmer               = "dimmer"
	function_measured_temperature = "measured_temperature"
	function_set_temperature      = "set_temperature"
	function_eco_mode             = "eco_mode"
)

var Functions = []string{
	// Note: Depends on order.
	function_dimmer,
	function_switch,
	function_measured_temperature,
	function_set_temperature,
	function_eco_mode,
}

func getAPI(config apiserver.Configuration) (*abb.Api, error) {
	var api *abb.Api
	if config.IsCloud {
		if config.ClientID == nil || config.ClientSecret == nil || config.RequestTimeout == nil {
			return nil, fmt.Errorf("one or more required config fields (ClientID, ClientSecret, RequestTimeout) are nil")
		}
		api = abb.NewGraphQLApi(config, "https://api.eu.mybuildings.abb.com", "https://api.eu.mybuildings.abb.com/external/oauth2helper/code/set/cd1a7768-680d-4040-ab76-b6a6f9c4bf9d")
	} else {
		if config.ApiUsername == nil || config.ApiPassword == nil || config.ApiUrl == nil || config.RequestTimeout == nil {
			return nil, fmt.Errorf("one or more required config fields (ApiUsername, ApiPassword, ApiUrl, RequestTimeout) are nil")
		}
		api = abb.NewLocalApi(*config.ApiUsername, *config.ApiPassword, *config.ApiUrl, int(*config.RequestTimeout))
	}
	if err := api.Authorize(); err != nil {
		if _, err := conf.InvalidateAuthorization(config); err != nil {
			return nil, fmt.Errorf("invalidating authorization: %v", err)
		}
		return nil, fmt.Errorf("authorizing: %v", err)
	}
	if _, err := conf.PersistAuthorization(&config, *api.Auth.OauthToken); err != nil {
		return nil, fmt.Errorf("persisting authorization: %v", err)
	}
	return api, nil
}

func GetSystems(config apiserver.Configuration) ([]model.System, error) {
	api, err := getAPI(config)
	if err != nil {
		return nil, fmt.Errorf("getting API instance: %v", err)
	}
	abbConfiguration, err := api.GetConfiguration()
	if err != nil {
		return nil, fmt.Errorf("getting configuration: %v", err)
	}

	var systems []model.System
	for id, system := range abbConfiguration.Systems {
		s := model.System{
			ID:   id,
			GAI:  id,
			Name: system.SysApName,
		}
		// fmt.Printf("system: %v\n", id)
		// fmt.Printf("ConnectionState: %v\n", system.ConnectionState)
		// fmt.Printf("Floorplan: %v\n", system.Floorplan)
		// fmt.Printf("SysAP: %v\n", system.SysApName)
		for id, device := range system.Devices {
			d := model.Device{
				ID:   id,
				GAI:  s.GAI + "_" + id,
				Name: device.DisplayName.(string),
			}
			// 	fmt.Printf("device: %v\n", id)
			// 	fmt.Printf("DeviceName: %v\n", device.DisplayName)
			// 	fmt.Printf("Floor: %v\n", device.Floor)
			// 	fmt.Printf("Room: %v\n", device.Room)
			// 	fmt.Printf("Interface: %v\n", device.Interface)
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
					NameBase: channel.DisplayName.(string) + " " + id,
				}
				switch fid {
				case abb.FID_SWITCH_ACTUATOR:
					switchStateStr := channel.FindOutputValueByPairingID(abb.PID_ON_OFF_INFO_GET)
					switchState, err := strconv.ParseInt(switchStateStr, 10, 8)
					if err != nil {
						log.Error("broker", "parsing switch output value '%s': %v", switchStateStr, err)
					}
					switchInputStr := channel.FindInputValueByPairingID(abb.PID_SWITCH_ON_OFF_SET)
					switchInput, err := strconv.ParseInt(switchInputStr, 10, 8)
					if err != nil {
						log.Error("broker", "parsing input value '%s': %v", switchInputStr, err)
					}

					outputs := make(map[string]model.Datapoint)
					for datapoint, input := range channel.Outputs {
						if input.PairingId == abb.PID_ON_OFF_INFO_GET {
							outputs[function_switch] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "switch_state",
									},
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "switch",
									},
								},
							}
						}
					}
					assetBase.OutputsBase = outputs
					inputs := make(map[string]string)
					for datapoint, input := range channel.Inputs {
						if input.PairingId == abb.PID_SWITCH_ON_OFF_SET {
							inputs[function_switch] = datapoint
						}
					}
					assetBase.InputsBase = inputs

					c = model.Switch{
						AssetBase:   assetBase,
						SwitchState: int8(switchState),
						Switch:      int8(switchInput),
					}
				case abb.FID_DIMMING_ACTUATOR:
					switchStateStr := channel.FindOutputValueByPairingID(abb.PID_ON_OFF_INFO_GET)
					switchState, err := strconv.ParseInt(switchStateStr, 10, 8)
					if err != nil {
						log.Error("broker", "parsing output value '%s': %v", switchStateStr, err)
					}
					switchInputStr := channel.FindInputValueByPairingID(abb.PID_SWITCH_ON_OFF_SET)
					switchInput, err := strconv.ParseInt(switchInputStr, 10, 8)
					if err != nil {
						log.Error("broker", "parsing input value '%s': %v", switchInputStr, err)
					}
					dimmerStateStr := channel.FindOutputValueByPairingID(abb.PID_ACTUAL_DIM_VALUE_0_100_GET)
					dimmerState, err := strconv.ParseInt(dimmerStateStr, 10, 8)
					if err != nil {
						log.Error("broker", "parsing output value '%s': %v", dimmerStateStr, err)
					}
					dimmerInputStr := channel.FindInputValueByPairingID(abb.PID_ABSOLUTE_VALUE_0_100_SET)
					dimmerInput, err := strconv.ParseInt(dimmerInputStr, 10, 8)
					if err != nil {
						log.Error("broker", "parsing input value '%s': %v", dimmerInputStr, err)
					}

					outputs := make(map[string]model.Datapoint)
					for datapoint, output := range channel.Outputs {
						switch output.PairingId {
						case abb.PID_ON_OFF_INFO_GET:
							outputs[function_switch] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "switch_state",
									},
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "switch",
									},
								},
							}
						case abb.PID_ACTUAL_DIM_VALUE_0_100_GET:
							outputs[function_dimmer] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "dimmer_state",
									},
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
						case abb.PID_SWITCH_ON_OFF_SET:
							inputs[function_switch] = datapoint
						case abb.PID_ABSOLUTE_VALUE_0_100_SET:
							inputs[function_dimmer] = datapoint
						}
					}
					assetBase.InputsBase = inputs

					c = model.Dimmer{
						AssetBase:   assetBase,
						SwitchState: int8(switchState),
						Switch:      int8(switchState),
						DimmerState: int8(dimmerState),
						Dimmer:      int8(dimmerState),
					}
					_, _ = dimmerInput, switchInput
				case abb.FID_ROOM_TEMPERATURE_CONTROLLER_MASTER_WITH_FAN, abb.FID_ROOM_TEMPERATURE_CONTROLLER_MASTER_WITHOUT_FAN, abb.FID_ROOM_TEMPERATURE_CONTROLLER_SLAVE:
					switchStateStr := channel.FindOutputValueByPairingID(abb.PID_CONTROLLER_ON_OFF_PROTECTED_GET)
					switchState, err := strconv.ParseInt(switchStateStr, 10, 8)
					if err != nil {
						log.Error("broker", "parsing output value '%s': %v", switchStateStr, err)
					}
					currentTempStr := channel.FindOutputValueByPairingID(abb.PID_MEASURED_TEMPERATURE)
					currentTemp, err := strconv.ParseFloat(currentTempStr, 16)
					if err != nil {
						log.Error("broker", "parsing output value '%s': %v", currentTempStr, err)
					}
					setTempStr := channel.FindOutputValueByPairingID(abb.PID_SETPOINT_TEMPERATURE_GET)
					setTemp, err := strconv.ParseFloat(setTempStr, 16)
					if err != nil {
						log.Error("broker", "parsing output value '%s': %v", setTempStr, err)
					}
					ecoModeStr := channel.FindOutputValueByPairingID(abb.PID_CONTROLLER_ECOMODE_SET)
					ecoMode, err := strconv.ParseInt(ecoModeStr, 10, 8)
					if err != nil {
						log.Error("broker", "parsing output value '%s': %v", ecoModeStr, err)
					}

					outputs := make(map[string]model.Datapoint)
					for datapoint, output := range channel.Outputs {
						switch output.PairingId {
						case abb.PID_CONTROLLER_ON_OFF_PROTECTED_GET:
							outputs[function_switch] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "switch_state",
									},
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "switch",
									},
								},
							}
						case abb.PID_MEASURED_TEMPERATURE:
							outputs[function_measured_temperature] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "current_temperature",
									},
								},
							}
						case abb.PID_SETPOINT_TEMPERATURE_GET:
							outputs[function_set_temperature] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "set_temperature_state",
									},
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "set_temperature",
									},
								},
							}
						case abb.PID_CONTROLLER_ECOMODE_SET:
							outputs[function_eco_mode] = model.Datapoint{
								Name: datapoint,
								Map: model.DatapointMap{
									{
										Subtype:       elionaapi.SUBTYPE_INPUT,
										AttributeName: "eco_mode_state",
									},
									{
										Subtype:       elionaapi.SUBTYPE_OUTPUT,
										AttributeName: "eco_mode",
									},
								},
							}
						}
					}
					assetBase.OutputsBase = outputs

					inputs := make(map[string]string)
					for datapoint, input := range channel.Inputs {
						switch input.PairingId {
						case abb.PID_CONTROLLER_REQ_ON_OFF_SET:
							inputs[function_switch] = datapoint
						case abb.PID_ABS_TEMPERATURE_SET:
							inputs[function_set_temperature] = datapoint
						case abb.PID_CONTROLLER_ECOMODE_SET:
							inputs[function_eco_mode] = datapoint
						}
					}
					assetBase.InputsBase = inputs

					c = model.RTC{
						AssetBase:    assetBase,
						SwitchState:  int8(switchState),
						Switch:       int8(switchState),
						CurrentTemp:  float32(currentTemp),
						SetTemp:      float32(setTemp),
						SetTempState: float32(setTemp),
						EcoMode:      int8(ecoMode),
						EcoModeState: int8(ecoMode),
					}
				default:
					c = model.Channel{
						AssetBase: assetBase,
					}
				}
				d.Channels = append(d.Channels, c)
				// fmt.Printf("channel: %v\n", id)
				// fmt.Printf("ChannelName: %v\n", channel.DisplayName)
				// fmt.Printf("FunctionId: %v\n", channel.FunctionId)
				// for id, input := range channel.Inputs {
				// 	fmt.Printf("InputID: %v\n", id)
				// 	fmt.Printf("InputPairingId: %v\n", input.PairingId)
				// 	fmt.Printf("InputValue: %v\n", input.Value)
				// }
				// for id, output := range channel.Outputs {
				// 	fmt.Printf("OutputID: %v\n", id)
				// 	fmt.Printf("OutputPairingId: %v\n", output.PairingId)
				// 	fmt.Printf("OutputValue: %v\n", output.Value)
				// }
			}
			s.Devices = append(s.Devices, d)
		}
		systems = append(systems, s)
	}
	return systems, nil
}

func ListenForDataChanges(config apiserver.Configuration, datapoints []appdb.Datapoint, ch chan<- abbgraphql.DataPoint) error {
	api, err := getAPI(config)
	if err != nil {
		return fmt.Errorf("getting API instance: %v", err)
	}
	if err := api.ListenGraphQLSubscriptions(datapoints, ch); err != nil {
		return fmt.Errorf("listen for graphQL subscriptions: %v", err)
	}
	return nil
}

func SetInput(config apiserver.Configuration, input appdb.Datapoint, value any) error {
	api, err := getAPI(config)
	if err != nil {
		return fmt.Errorf("getting API instance: %v", err)
	}
	return api.WriteDatapoint(input.SystemID, input.DeviceID, input.ChannelID, input.Datapoint, value)
}
