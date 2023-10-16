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

package eliona

import (
	"fmt"

	api "github.com/eliona-smart-building-assistant/go-eliona-api-client/v2"
	"github.com/eliona-smart-building-assistant/go-eliona/client"
	"github.com/eliona-smart-building-assistant/go-utils/common"
)

func GetDashboard(projectId string) (api.Dashboard, error) {
	dashboard := api.Dashboard{}
	dashboard.Name = "ABB-free@home"
	dashboard.ProjectId = projectId
	dashboard.Widgets = []api.Widget{}

	rootAssets, _, err := client.NewClient().AssetsAPI.
		GetAssets(client.AuthenticationContext()).
		AssetTypeName("abb_free_at_home_root").
		ProjectId(projectId).
		Execute()
	if err != nil {
		return api.Dashboard{}, fmt.Errorf("fetching root asset: %v", err)
	}
	if len(rootAssets) != 1 {
		return api.Dashboard{}, fmt.Errorf("found %v root assets in project %v, expected 1", len(rootAssets), projectId)
	}
	rootAsset := rootAssets[0]

	switches, _, err := client.NewClient().AssetsAPI.
		GetAssets(client.AuthenticationContext()).
		AssetTypeName("abb_free_at_home_switch_sensor").
		ProjectId(projectId).
		Execute()
	if err != nil {
		return api.Dashboard{}, fmt.Errorf("fetching switches: %v", err)
	}
	widgetSequence := int32(0)
	var switchesData []api.WidgetData
	for i, sw := range switches {
		switchesData = append(switchesData, api.WidgetData{
			ElementSequence: nullableInt32(1),
			AssetId:         sw.Id,
			Data: map[string]interface{}{
				"attribute":   "switch",
				"description": sw.Name,
				"key":         "_CURRENT",
				"seq":         i,
				"subtype":     "output",
			},
		})
		switchesData = append(switchesData, api.WidgetData{
			ElementSequence: nullableInt32(1),
			AssetId:         sw.Id,
			Data: map[string]interface{}{
				"attribute":   "switch",
				"description": sw.Name,
				"key":         "_SETPOINT",
				"seq":         i,
				"subtype":     "output",
			},
		})
	}
	widget := api.Widget{
		WidgetTypeName: "ABB Switch list",
		AssetId:        rootAsset.Id,
		Sequence:       nullableInt32(widgetSequence),
		Details: map[string]any{
			"size":     1,
			"timespan": 7,
		},
		Data: switchesData,
	}
	widgetSequence++
	dashboard.Widgets = append(dashboard.Widgets, widget)

	dimmers, _, err := client.NewClient().AssetsAPI.
		GetAssets(client.AuthenticationContext()).
		AssetTypeName("abb_free_at_home_dimmer_sensor").
		ProjectId(projectId).
		Execute()
	if err != nil {
		return api.Dashboard{}, fmt.Errorf("fetching dimmers: %v", err)
	}
	var dimmersData []api.WidgetData
	for i, d := range dimmers {
		dimmersData = append(dimmersData, api.WidgetData{
			ElementSequence: nullableInt32(1),
			AssetId:         d.Id,
			Data: map[string]interface{}{
				"attribute":   "dimmer",
				"description": d.Name,
				"key":         "_CURRENT",
				"seq":         i,
				"subtype":     "output",
			},
		})
		dimmersData = append(dimmersData, api.WidgetData{
			ElementSequence: nullableInt32(1),
			AssetId:         d.Id,
			Data: map[string]interface{}{
				"attribute":   "dimmer",
				"description": d.Name,
				"key":         "_SETPOINT",
				"seq":         i,
				"subtype":     "output",
			},
		})
	}
	widget = api.Widget{
		WidgetTypeName: "ABB Dimmer list",
		AssetId:        rootAsset.Id,
		Sequence:       nullableInt32(widgetSequence),
		Details: map[string]any{
			"size":     1,
			"timespan": 7,
		},
		Data: dimmersData,
	}
	widgetSequence++
	dashboard.Widgets = append(dashboard.Widgets, widget)

	hueLights, _, err := client.NewClient().AssetsAPI.
		GetAssets(client.AuthenticationContext()).
		AssetTypeName("abb_free_at_home_hue_actuator").
		ProjectId(projectId).
		Execute()
	if err != nil {
		return api.Dashboard{}, fmt.Errorf("fetching hueLights: %v", err)
	}

	for _, hueLight := range hueLights {
		dashboard.Widgets = append(dashboard.Widgets, api.Widget{
			WidgetTypeName: "ABB Philips Hue",
			AssetId:        hueLight.Id,
			Sequence:       nullableInt32(widgetSequence),
			Details: map[string]any{
				"size":     1,
				"timespan": 7,
			},
			Data: []api.WidgetData{
				{
					ElementSequence: nullableInt32(1),
					AssetId:         hueLight.Id,
					Data: map[string]interface{}{
						"attribute":   "switch",
						"description": "Light",
						"key":         "_SETPOINT",
						"seq":         0,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(1),
					AssetId:         hueLight.Id,
					Data: map[string]interface{}{
						"attribute":   "switch",
						"description": "Light",
						"key":         "_CURRENT",
						"seq":         0,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         hueLight.Id,
					Data: map[string]interface{}{
						"attribute":   "dimmer",
						"description": "Dimmer",
						"key":         "_SETPOINT",
						"seq":         0,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         hueLight.Id,
					Data: map[string]interface{}{
						"attribute":   "dimmer",
						"description": "Dimmer",
						"key":         "_CURRENT",
						"seq":         0,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         hueLight.Id,
					Data: map[string]interface{}{
						"attribute":   "color_temperature",
						"description": "Color temperature",
						"key":         "_CURRENT",
						"seq":         1,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         hueLight.Id,
					Data: map[string]interface{}{
						"attribute":   "color_temperature",
						"description": "Color temperature",
						"key":         "_SETPOINT",
						"seq":         1,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         hueLight.Id,
					Data: map[string]interface{}{
						"attribute":   "hsv_state",
						"description": "Hue",
						"key":         "_CURRENT",
						"seq":         2,
						"subtype":     "input",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         hueLight.Id,
					Data: map[string]interface{}{
						"attribute":   "hsv_hue",
						"description": "Hue",
						"key":         "_SETPOINT",
						"seq":         2,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         hueLight.Id,
					Data: map[string]interface{}{
						"attribute":   "hsv_saturation",
						"description": "Saturation",
						"key":         "_SETPOINT",
						"seq":         3,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         hueLight.Id,
					Data: map[string]interface{}{
						"attribute":   "color_mode_state",
						"description": "Saturation",
						"key":         "_CURRENT",
						"seq":         3,
						"subtype":     "input",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         hueLight.Id,
					Data: map[string]interface{}{
						"attribute":   "hsv_value",
						"description": "Value",
						"key":         "_SETPOINT",
						"seq":         4,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         hueLight.Id,
					Data: map[string]interface{}{
						"attribute":   "switch_state",
						"description": "Value",
						"key":         "_CURRENT",
						"seq":         4,
						"subtype":     "input",
					},
				},
			},
		})
		widgetSequence++
	}

	movementSensors, _, err := client.NewClient().AssetsAPI.
		GetAssets(client.AuthenticationContext()).
		AssetTypeName("abb_free_at_home_movement_sensor").
		ProjectId(projectId).
		Execute()
	if err != nil {
		return api.Dashboard{}, fmt.Errorf("fetching movementSensors: %v", err)
	}

	for _, movementSensor := range movementSensors {
		dashboard.Widgets = append(dashboard.Widgets, api.Widget{
			WidgetTypeName: "ABB Binary trend",
			AssetId:        movementSensor.Id,
			Sequence:       nullableInt32(widgetSequence),
			Details: map[string]any{
				"size":     1,
				"timespan": 7,
			},
			Data: []api.WidgetData{
				{
					ElementSequence: nullableInt32(1),
					AssetId:         movementSensor.Id,
					Data: map[string]interface{}{
						"aggregatedDataType": "heap",
						"attribute":          "movement",
						"description":        "Movement detector ",
						"key":                "",
						"seq":                0,
						"subtype":            "input",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         movementSensor.Id,
					Data: map[string]interface{}{
						"aggregatedDataField":  nil,
						"aggregatedDataRaster": nil,
						"aggregatedDataType":   "heap",
						"attribute":            "movement",
						"description":          "Movement detector ",
						"key":                  "",
						"seq":                  0,
						"subtype":              "input",
					},
				},
			},
		})
		widgetSequence++
	}

	rtcs, _, err := client.NewClient().AssetsAPI.
		GetAssets(client.AuthenticationContext()).
		AssetTypeName("abb_free_at_home_room_temperature_controller").
		ProjectId(projectId).
		Execute()
	if err != nil {
		return api.Dashboard{}, fmt.Errorf("fetching rtcs: %v", err)
	}

	for _, rtc := range rtcs {
		dashboard.Widgets = append(dashboard.Widgets, api.Widget{
			WidgetTypeName: "ABB Temperature regulator",
			AssetId:        rtc.Id,
			Sequence:       nullableInt32(widgetSequence),
			Details: map[string]interface{}{
				"2006": map[string]interface{}{
					"colors": []string{
						"#656565",
					},
					"guideline": map[string]interface{}{
						"type":  "value",
						"value": "20",
					},
				},
				"size":     1,
				"timespan": 7,
			},
			Data: []api.WidgetData{
				{
					ElementSequence: nullableInt32(1),
					AssetId:         rtc.Id,
					Data: map[string]interface{}{
						"attribute":   "switch",
						"description": "Room temperature controller",
						"key":         "_SETPOINT",
						"seq":         0,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(1),
					AssetId:         rtc.Id,
					Data: map[string]interface{}{
						"attribute":   "switch",
						"description": "Room temperature controller",
						"key":         "_CURRENT",
						"seq":         0,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         rtc.Id,
					Data: map[string]interface{}{
						"aggregatedDataField":  nil,
						"aggregatedDataRaster": nil,
						"aggregatedDataType":   "heap",
						"attribute":            "current_temperature",
						"description":          "Temperature",
						"key":                  "",
						"seq":                  0,
						"subtype":              "input",
					},
				},
				{
					ElementSequence: nullableInt32(3),
					AssetId:         rtc.Id,
					Data: map[string]interface{}{
						"attribute":   "set_temperature",
						"description": "Desired temperature",
						"key":         "_CURRENT",
						"seq":         0,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(3),
					AssetId:         rtc.Id,
					Data: map[string]interface{}{
						"attribute":   "set_temperature",
						"description": "Desired temperature",
						"key":         "_SETPOINT",
						"seq":         0,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(4),
					AssetId:         rtc.Id,
					Data: map[string]interface{}{
						"aggregatedDataField":  nil,
						"aggregatedDataRaster": nil,
						"aggregatedDataType":   "heap",
						"attribute":            "current_temperature",
						"description":          "Current temperature",
						"key":                  "",
						"seq":                  0,
						"subtype":              "input",
					},
				},
			},
		})
		widgetSequence++
	}

	thermostats, _, err := client.NewClient().AssetsAPI.
		GetAssets(client.AuthenticationContext()).
		AssetTypeName("abb_free_at_home_radiator_thermostat").
		ProjectId(projectId).
		Execute()
	if err != nil {
		return api.Dashboard{}, fmt.Errorf("fetching thermostats: %v", err)
	}

	for _, thermostat := range thermostats {
		dashboard.Widgets = append(dashboard.Widgets, api.Widget{
			WidgetTypeName: "ABB Temperature regulator",
			AssetId:        thermostat.Id,
			Sequence:       nullableInt32(4),
			Details: map[string]interface{}{
				"size":     1,
				"timespan": 7,
				"2006": map[string]interface{}{
					"barValues":     []interface{}{},
					"colors":        []string{"#656565"},
					"description":   "",
					"guideline":     map[string]interface{}{"type": "value", "value": "21"},
					"multipleYAxes": false,
					"overlap":       true,
					"showCurrent":   true,
					"type":          "analog",
					"variant":       "line",
					"yAxisLabels":   "mam",
				},
			},
			Data: []api.WidgetData{
				{
					ElementSequence: nullableInt32(1),
					AssetId:         thermostat.Id,
					Data: map[string]interface{}{
						"attribute":   "switch",
						"description": "Radiator thermostate",
						"key":         "_SETPOINT",
						"seq":         0,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(1),
					AssetId:         thermostat.Id,
					Data: map[string]interface{}{
						"attribute":   "switch",
						"description": "Radiator thermostate",
						"key":         "_CURRENT",
						"seq":         0,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         thermostat.Id,
					Data: map[string]interface{}{
						"aggregatedDataField":  nil,
						"aggregatedDataRaster": nil,
						"aggregatedDataType":   "heap",
						"attribute":            "current_temperature",
						"description":          "Temperature",
						"key":                  "",
						"seq":                  0,
						"subtype":              "input",
					},
				},
				{
					ElementSequence: nullableInt32(3),
					AssetId:         thermostat.Id,
					Data: map[string]interface{}{
						"attribute":   "set_temperature",
						"description": "Desired temperature",
						"key":         "_CURRENT",
						"seq":         0,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(3),
					AssetId:         thermostat.Id,
					Data: map[string]interface{}{
						"attribute":   "set_temperature",
						"description": "Desired temperature",
						"key":         "_SETPOINT",
						"seq":         0,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(4),
					AssetId:         thermostat.Id,
					Data: map[string]interface{}{
						"aggregatedDataField":  nil,
						"aggregatedDataRaster": nil,
						"aggregatedDataType":   "heap",
						"attribute":            "current_temperature",
						"description":          "Current temperature",
						"key":                  "",
						"seq":                  0,
						"subtype":              "input",
					},
				},
			},
		})
		widgetSequence++
	}

	windowSensors, _, err := client.NewClient().AssetsAPI.
		GetAssets(client.AuthenticationContext()).
		AssetTypeName("abb_free_at_home_window_sensor").
		ProjectId(projectId).
		Execute()
	if err != nil {
		return api.Dashboard{}, fmt.Errorf("fetching windowSensors: %v", err)
	}

	for _, windowSensor := range windowSensors {
		dashboard.Widgets = append(dashboard.Widgets, api.Widget{
			WidgetTypeName: "ABB Window sensor",
			AssetId:        windowSensor.Id,
			Sequence:       nullableInt32(6),
			Details: map[string]interface{}{
				"size":     1,
				"timespan": 7,
				"2007": map[string]interface{}{
					"tilesConfig": []map[string]interface{}{
						{
							"defaultColorIndex": 7,
							"progressBar":       nil,
							"valueMapping":      []interface{}{},
						},
					},
				},
				"2008": map[string]interface{}{
					"colors": []string{"#656565"},
				},
			},
			Data: []api.WidgetData{
				{
					ElementSequence: nullableInt32(1),
					AssetId:         windowSensor.Id,
					Data: map[string]interface{}{
						"aggregatedDataType": "heap",
						"attribute":          "position",
						"description":        "Window sensor",
						"key":                "",
						"seq":                0,
						"subtype":            "input",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         windowSensor.Id,
					Data: map[string]interface{}{
						"aggregatedDataField":  nil,
						"aggregatedDataRaster": nil,
						"aggregatedDataType":   "heap",
						"attribute":            "position",
						"description":          "Window sensor",
						"key":                  "",
						"seq":                  0,
						"subtype":              "input",
					},
				},
			},
		})
		widgetSequence++
	}

	doorSensors, _, err := client.NewClient().AssetsAPI.
		GetAssets(client.AuthenticationContext()).
		AssetTypeName("abb_free_at_home_door_sensor").
		ProjectId(projectId).
		Execute()
	if err != nil {
		return api.Dashboard{}, fmt.Errorf("fetching doorSensors: %v", err)
	}

	for _, doorSensor := range doorSensors {
		dashboard.Widgets = append(dashboard.Widgets, api.Widget{
			WidgetTypeName: "ABB Binary trend",
			AssetId:        doorSensor.Id,
			Sequence:       nullableInt32(7),
			Details: map[string]interface{}{
				"size":     1,
				"timespan": 7,
				"2001": map[string]interface{}{
					"tilesConfig": []map[string]interface{}{
						{
							"defaultColorIndex": 7,
							"progressBar":       nil,
							"valueMapping":      []interface{}{},
						},
					},
				},
				"2002": map[string]interface{}{
					"colors": []string{"#656565"},
				},
			},
			Data: []api.WidgetData{
				{
					ElementSequence: nullableInt32(1),
					AssetId:         doorSensor.Id,
					Data: map[string]interface{}{
						"aggregatedDataType": "heap",
						"attribute":          "position",
						"description":        "Door/window contact ",
						"key":                "",
						"seq":                0,
						"subtype":            "input",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         doorSensor.Id,
					Data: map[string]interface{}{
						"aggregatedDataField":  nil,
						"aggregatedDataRaster": nil,
						"aggregatedDataType":   "heap",
						"attribute":            "position",
						"description":          "Door/window contact ",
						"key":                  "",
						"seq":                  0,
						"subtype":              "input",
					},
				},
			},
		})
		widgetSequence++
	}

	return dashboard, nil
}

func nullableInt32(val int32) api.NullableInt32 {
	return *api.NewNullableInt32(common.Ptr[int32](val))
}
