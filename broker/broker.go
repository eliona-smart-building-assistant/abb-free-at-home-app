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
	"abb-free-at-home/apiserver"
	"abb-free-at-home/appdb"
	"abb-free-at-home/conf"
	"fmt"
	"reflect"
	"strconv"

	"github.com/eliona-smart-building-assistant/go-utils/common"
	"github.com/eliona-smart-building-assistant/go-utils/log"
)

const (
	function_switch = "switch"
	function_dimmer = "dimmer"
)

var Functions = []string{
	// Note: Depends on order.
	function_dimmer,
	function_switch,
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
	id     string `eliona:"channel_id,filterable"`
	gai    string
	name   string `eliona:"channel_name,filterable"`
	inputs map[string]string
}

func (a AssetBase) Name() string {
	return a.name
}

func (a AssetBase) Id() string {
	return a.id
}

func (a AssetBase) Inputs() map[string]string {
	return a.inputs
}

type Channel struct {
	AssetBase
}

func (c Channel) AssetType() string {
	return "abb_free_at_home_channel"
}

func (c Channel) GAI() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.gai)
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
	return fmt.Sprintf("%s_%s", c.AssetType(), c.gai)
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
	return fmt.Sprintf("%s_%s", c.AssetType(), c.gai)
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
		return nil, fmt.Errorf("authorizing: %v", err)
	}
	if _, err := conf.PersistAuthorization(config, *api.Auth.OauthToken); err != nil {
		return nil, fmt.Errorf("persisting authorization: %v", err)
	}
	return api, nil
}

func GetSystems(config apiserver.Configuration) ([]System, error) {
	api, err := getAPI(config)
	if err != nil {
		return nil, fmt.Errorf("getting API instance: %v", err)
	}
	abbConfiguration, err := api.GetConfiguration()
	if err != nil {
		return nil, fmt.Errorf("getting configuration: %v", err)
	}

	var systems []System
	for id, system := range abbConfiguration.Systems {
		s := System{
			ID:   id,
			GAI:  id,
			Name: system.SysApName,
		}
		// fmt.Printf("system: %v\n", id)
		// fmt.Printf("ConnectionState: %v\n", system.ConnectionState)
		// fmt.Printf("Floorplan: %v\n", system.Floorplan)
		// fmt.Printf("SysAP: %v\n", system.SysApName)
		for id, device := range system.Devices {
			d := Device{
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
				var c Asset
				fid, err := strconv.ParseInt(channel.FunctionId, 16, 0)
				if err != nil {
					log.Error("broker", "parsing functionID %s: %v", channel.FunctionId, err)
					continue
				}
				assetBase := AssetBase{
					id:   id,
					gai:  d.GAI + "_" + id,
					name: channel.DisplayName.(string) + " " + id,
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
					inputs := make(map[string]string)
					for datapoint, input := range channel.Inputs {
						if input.PairingId == abb.PID_SWITCH_ON_OFF_SET {
							inputs[function_switch] = datapoint
						}
					}
					assetBase.inputs = inputs
					c = Switch{
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
					inputs := make(map[string]string)
					for datapoint, input := range channel.Inputs {
						switch input.PairingId {
						case abb.PID_SWITCH_ON_OFF_SET:
							inputs[function_switch] = datapoint
						case abb.PID_ABSOLUTE_VALUE_0_100_SET:
							inputs[function_dimmer] = datapoint
						}
					}
					assetBase.inputs = inputs
					c = Dimmer{
						AssetBase:   assetBase,
						SwitchState: int8(switchState),
						Switch:      int8(switchState),
						DimmerState: int8(dimmerState),
						Dimmer:      int8(dimmerState),
					}
					_, _ = dimmerInput, switchInput
				default:
					c = Channel{
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

func SetInput(config apiserver.Configuration, output appdb.Input, value any) error {
	api, err := getAPI(config)
	if err != nil {
		return fmt.Errorf("getting API instance: %v", err)
	}
	return api.WriteDatapoint(output.SystemID, output.DeviceID, output.ChannelID, output.Datapoint, value)
}
