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
	sequence := int32(0)
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
		WidgetTypeName: "Switch",
		AssetId:        rootAsset.Id,
		Sequence:       nullableInt32(sequence),
		Details: map[string]any{
			"size":     1,
			"timespan": 7,
		},
		Data: switchesData,
	}
	sequence++
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
		WidgetTypeName: "LightControl",
		AssetId:        rootAsset.Id,
		Sequence:       nullableInt32(sequence),
		Details: map[string]any{
			"size":     1,
			"timespan": 7,
		},
		Data: dimmersData,
	}
	sequence++
	dashboard.Widgets = append(dashboard.Widgets, widget)

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
			WidgetTypeName: "AnalogInput",
			AssetId:        rtc.Id,
			Sequence:       nullableInt32(sequence),
			Details: map[string]any{
				"5": map[string]any{
					"colors": []string{
						"#61D583",
						"#35c7d5",
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
						"aggregatedDataType": "heap",
						"attribute":          "current_temperature",
						"description":        "Current temperature",
						"key":                "",
						"seq":                0,
						"subtype":            "input",
					},
				},
				{
					ElementSequence: nullableInt32(1),
					AssetId:         rtc.Id,
					Data: map[string]interface{}{
						"aggregatedDataType": "heap",
						"attribute":          "set_temperature_state",
						"description":        "Set temperature",
						"key":                "",
						"seq":                1,
						"subtype":            "input",
					},
				},
				{
					ElementSequence: nullableInt32(2),
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
					ElementSequence: nullableInt32(2),
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
					ElementSequence: nullableInt32(2),
					AssetId:         rtc.Id,
					Data: map[string]interface{}{
						"attribute":   "eco_mode",
						"description": "ECO mode",
						"key":         "_CURRENT",
						"seq":         1,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(2),
					AssetId:         rtc.Id,
					Data: map[string]interface{}{
						"attribute":   "eco_mode",
						"description": "ECO mode",
						"key":         "_SETPOINT",
						"seq":         1,
						"subtype":     "output",
					},
				},
				{
					ElementSequence: nullableInt32(3),
					AssetId:         rtc.Id,
					Data: map[string]interface{}{
						"aggregatedDataType": "heap",
						"attribute":          "current_temperature",
						"description":        "Current temperature",
						"key":                "",
						"seq":                0,
						"subtype":            "input",
					},
				},
				{
					ElementSequence: nullableInt32(3),
					AssetId:         rtc.Id,
					Data: map[string]interface{}{
						"aggregatedDataType": "heap",
						"attribute":          "set_temperature_state",
						"description":        "Set temperature",
						"key":                "",
						"seq":                1,
						"subtype":            "input",
					},
				},
			},
		})
		sequence++
	}
	return dashboard, nil
}

func nullableInt32(val int32) api.NullableInt32 {
	return *api.NewNullableInt32(common.Ptr[int32](val))
}
