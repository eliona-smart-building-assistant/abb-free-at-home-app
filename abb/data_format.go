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
	PairingId int `json:"pairingID"`
}
type Output struct {
	Value     string `json:"value"`
	PairingId int    `json:"pairingID"`
}
type Channel struct {
	DisplayName interface{}       `json:"displayName"`
	Floor       interface{}       `json:"floor"`
	Inputs      map[string]Input  `json:"inputs"`
	Outputs     map[string]Output `json:"outputs"`
	FunctionId  string            `json:"functionID"`
	Room        interface{}       `json:"room"`
}
type Device struct {
	Channels    map[string]Channel `json:"channels"`
	DisplayName interface{}        `json:"displayName"`
	Floor       interface{}        `json:"floor"`
	Interface   interface{}        `json:"interface"`
	Room        interface{}        `json:"room"`
	Location    string
}
type System struct {
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
	Systems map[string]System `json:""`
}

func (c *Channel) FindOutputValueByPairingID(pairingId int) string {
	for _, o := range c.Outputs {
		if o.PairingId == pairingId {
			return o.Value
		}
	}
	return ""
}

func WsFormatToApiFormat(wsFormat *WsObject) map[string]System {
	dataFormat := DataFormat{
		Systems: map[string]System{},
	}
	t := System{
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
				return dataFormat.Systems
			}

			if strings.Contains(assignmentSplit[2], "odp") {
				o := Output{
					Value: value.(string),
				}
				c.Outputs[assignmentSplit[2]] = o
			} else {
				i := Input{}
				c.Inputs[assignmentSplit[2]] = i
			}

			d.Channels[assignmentSplit[1]] = c
			t.Devices[assignmentSplit[0]] = d

			dataFormat.Systems[tentant] = t
		}
	}

	return dataFormat.Systems
}
