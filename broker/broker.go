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
	"fmt"
	"reflect"

	"github.com/eliona-smart-building-assistant/go-utils/common"
)

type Asset interface {
	AssetType() string
	Id() string
	Name() string
}

type System struct {
	ID      string `eliona:"system_id,filterable"`
	Name    string `eliona:"system_name,filterable"`
	Devices []Device
}

type Device struct {
	ID       string `eliona:"device_id,filterable"`
	Name     string `eliona:"device_name,filterable"`
	Channels []Asset
}

type Channel struct {
	id   string `eliona:"channel_id,filterable"`
	name string `eliona:"channel_name,filterable"`
}

func (c Channel) AssetType() string {
	return "abb_free_at_home_channel"
}

func (c Channel) Id() string {
	return fmt.Sprintf("%s_%s", c.AssetType(), c.id)
}

func (c Channel) Name() string {
	return c.name
}

func GetSystems(config apiserver.Configuration) ([]System, error) {
	api := abb.NewLocalApi(config.ApiUsername, config.ApiPassword, config.ApiUrl, int(*config.RequestTimeout))
	abbConfiguration, err := api.GetConfiguration()
	if err != nil {
		return nil, fmt.Errorf("getting configuration: %v", err)
	}

	var systems []System
	for id, system := range abbConfiguration.Systems {
		s := System{
			ID:   id,
			Name: system.SysApName,
		}
		// fmt.Printf("system: %v\n", id)
		// fmt.Printf("ConnectionState: %v\n", system.ConnectionState)
		// fmt.Printf("Floorplan: %v\n", system.Floorplan)
		// fmt.Printf("SysAP: %v\n", system.SysApName)
		for id, device := range system.Devices {
			d := Device{
				ID:   s.ID + "_" + id,
				Name: device.DisplayName.(string),
			}
			// 	fmt.Printf("device: %v\n", id)
			// 	fmt.Printf("DeviceName: %v\n", device.DisplayName)
			// 	fmt.Printf("Floor: %v\n", device.Floor)
			// 	fmt.Printf("Room: %v\n", device.Room)
			// 	fmt.Printf("Interface: %v\n", device.Interface)
			for id, channel := range device.Channels {
				c := Channel{
					ID:   d.ID + "_" + id,
					Name: channel.DisplayName.(string) + " " + id,
				}
				d.Channels = append(d.Channels, c)
				fmt.Printf("channel: %v\n", id)
				fmt.Printf("ChannelName: %v\n", channel.DisplayName)
				fmt.Printf("FunctionId: %v\n", channel.FunctionId)
				for _, input := range channel.Inputs {
					fmt.Printf("OutputPairingId: %v\n", input.PairingId)
					fmt.Printf("OutputValue: %v\n", input.Value)
				}
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
