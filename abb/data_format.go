package abb

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

// Original author: Christian Stauffer <christian.stauffer@leicom.ch>

import (
	"log"
	"strings"
)

type WsObject map[string]struct {
	BadConfig       bool                   `json:"configDirty"`
	ConnectionState string                 `json:"connectionState"`
	DataPoints      map[string]interface{} `json:"datapoints"`
	Devices         interface{}            `json:"devices"`
	DevicesAdded    interface{}            `json:"devicesAdded"`
	DevicesRemoved  interface{}            `json:"devicesRemoved"`
	ScenesTriggered interface{}            `json:"scenesTriggered"`
	Timestamp       string                 `json:"timestamp"`
}

type Input struct {
	Value     string      `json:"value"`
	PairingId interface{} `json:"pairingID"`
}
type Output struct {
	Value     string      `json:"value"`
	PairingId interface{} `json:"pairingID"`
}
type Channel struct {
	DisplayName interface{}       `json:"displayName"`
	Floor       interface{}       `json:"floor"`
	Inputs      map[string]Input  `json:"inputs"`
	Outputs     map[string]Output `json:"outputs"`
	FunctionId  interface{}       `json:"functionID"`
	Room        interface{}       `json:"room"`
}
type Device struct {
	Channels    map[string]Channel `json:"channels"`
	DisplayName interface{}        `json:"displayName"`
	Floor       interface{}        `json:"floor"`
	Interface   interface{}        `json:"interface"`
	Room        interface{}        `json:"room"`
}
type Tentant struct {
	ConnectionState string            `json:"connectionState"`
	Devices         map[string]Device `json:"devices"`
	SysApName       string            `json:"sysapName"`
	Floorplan       Floors            `json:"floorplan"`
}
type Floors struct {
	Floors map[string]Floor `json:"floors"`
}
type Floor struct {
	Name    string `json:"name"`
	AssetId interface{}
	Rooms   map[string]Room `json:"rooms"`
}
type Room struct {
	Name    string `json:"name"`
	AssetId interface{}
}
type DataFormat struct {
	Tentants map[string]Tentant `json:""`
}

// function id description (yes, abb send's the id as string)
const (
	FID_SWITCH_SENSOR                                  = 0x0000
	FID_DIMMING_SENSOR                                 = 0x0001
	FID_BLIND_SENSOR                                   = 0x0003
	FID_SWITCH_ACTUATOR                                = 0x0007
	FID_SHUTTER_ACTUATOR                               = 0x0009
	FID_ROOM_TEMPERATURE_CONTROLLER_MASTER_WITH_FAN    = 0x000A
	FID_ROOM_TEMPERATURE_CONTROLLER_SLAVE              = 0x000B
	FID_DIMMING_ACTUATOR                               = 0x0012
	FID_ROOM_TEMPERATURE_CONTROLLER_MASTER_WITHOUT_FAN = 0x0023
	FID_BLIND_ACTUATOR                                 = 0x0061
	FID_ATTIC_WINDOW_ACTUATOR                          = 0x0062
	FID_AWNING_ACTUATOR                                = 0x0063
	FID_SMOKE_DETECTOR                                 = 0x007d
	FID_LightGroup                                     = 0x4000
	FID_DimmerGroup                                    = 0x4001
	FID_TimerProgramSwitchActuator                     = 0x4A00
	FID_Scence                                         = 0x4a01
	FID_DES_LEVEL_CALL_SENSOR                          = 0x001e
	FID_DES_LEVEL_CALL_ACTUATOR                        = 0x001D
	FID_DES_AUTOMATIC_DOOR_OPENER_ACTUATOR             = 0x0020
	FID_DES_DOOR_OPENER_ACTUATOR                       = 0x001A
	FID_DES_LIGHT_SWITCH_ACTUATOR                      = 0x0021
	FID_PANEL_SWITCH_SENSOR                            = 0x0030
	FID_PANEL_BLIND_SENSOR                             = 0x0033
	FID_PANEL_SCENE_SENSOR                             = 0x0037
	FID_DES_DOOR_RINGING_SENSOR                        = 0x001F
	FID_PANEL_ROOM_TEMPERATURE_CONTROLLER_SLAVE        = 0x0038
	FID_HEATING_ACTUATOR                               = 0x0027
	FID_COOLING_ACTUATOR                               = 0x0024
	FID_SCENE_SENSOR                                   = 0x0006
	FID_MOVEMENT_DETECTOR                              = 0x0011
)

// *************************** VALUE MAP IN ELIONA
// 0 - up
// 1 - down
// 0x01 - comfort mode
// 0x02 - standby
// 0x04 - eco mode
// 0x08 - building protect
// 0x10 - dew alarm
// 0x20 - heat (set) / cool (unset)
// 0x40 - no heating/cooling (set)
// 0x80 - frost alarm

// 0 - off (Protection mode)
// 1 - on (Comfort mode or Eco mode)

// 0 - Off
// 1 - On

// 0 - not moving
// 2 - moves up
// 3 - moves down

const (
	PID_SWITCH_ON_OFF_SET                          = 0x0001
	PID_TIMED_START_STOP                           = 0x0002
	PID_AL_RELATIVE_SET_VALUE_CONTROL              = 0x0010
	PID_ABSOLUTE_VALUE_0_100_SET                   = 0x0011
	PID_BLINDER_UP_DOWN_SET                        = 0x0020
	PID_BLINDER_STOP_SET                           = 0x0021
	PID_BLINDER_ABS_POSITION_0_100_SET             = 0x0023
	PID_SET_ABSOLUTE_POSITION_SLATS_PERCENTAGE     = 0x0024
	PID_ACTUATING_VALUE_HEATING                    = 0x0030
	PID_SETPOINT_TEMPERATURE_GET                   = 0x0033
	PID_HEATING_MODE_GET                           = 0x0036
	PID_CONTROLLER_ON_OFF_PROTECTED_GET            = 0x0038
	PID_CONTROLLER_ECOMODE_SET                     = 0x003A
	PID_CONTROLLER_REQ_ON_OFF_SET                  = 0x0042
	PID_ON_OFF_INFO_GET                            = 0x0100
	PID_ACTUAL_DIM_VALUE_0_100_GET                 = 0x0110
	PID_UP_DOWN_STOP_STATE                         = 0x0120
	PID_CURRENT_POSITION_BLIND_0_100_GET           = 0x0121
	PID_CURRENT_ABSOLUTE_POSITION_SLATS_PERCENTAGE = 0x0122
	PID_MEASURED_TEMPERATURE                       = 0x0130 // indicator ??
	PID_ABS_TEMPERATURE_SET                        = 0x0140 // set reg temp on room controller
)

func WsFormatToApiFormat(wsFormat *WsObject) map[string]Tentant {
	dataFormat := DataFormat{
		Tentants: map[string]Tentant{},
	}
	t := Tentant{
		Devices: map[string]Device{},
	}
	d := Device{
		Channels: map[string]Channel{},
	}
	c := Channel{
		Inputs:  map[string]Input{},
		Outputs: map[string]Output{},
	}

	for tentant, data := range *wsFormat {
		for assignment, value := range data.DataPoints {
			assignmentSplit := strings.Split(assignment, "/")

			// log.Printf("tentant: %s, assignment: %v, value: %v", tentant, assignmentSplit, value)

			if len(assignmentSplit) != 3 {
				log.Println("not matching len")
				return dataFormat.Tentants
			}

			if strings.Contains(assignmentSplit[2], "odp") {
				o := Output{
					Value: value.(string),
				}
				c.Outputs[assignmentSplit[2]] = o
			} else {
				i := Input{
					Value: value.(string),
				}
				c.Inputs[assignmentSplit[2]] = i
			}

			d.Channels[assignmentSplit[1]] = c
			t.Devices[assignmentSplit[0]] = d

			dataFormat.Tentants[tentant] = t
		}
	}

	return dataFormat.Tentants
}
