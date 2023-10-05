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
	"abb-free-at-home/apiserver"
	"abb-free-at-home/conf"
	"abb-free-at-home/model"
	"context"
	"fmt"

	api "github.com/eliona-smart-building-assistant/go-eliona-api-client/v2"
	"github.com/eliona-smart-building-assistant/go-eliona/asset"
	"github.com/eliona-smart-building-assistant/go-utils/common"
	"github.com/eliona-smart-building-assistant/go-utils/log"
)

type Asset interface {
	AssetType() string
	Id() string
}

func CreateLocationAssetsIfNecessary(config apiserver.Configuration, locations []model.Floor) error {
	for _, projectId := range conf.ProjIds(config) {
		rootAssetID, err := upsertRootAsset(config, projectId)
		if err != nil {
			return fmt.Errorf("upserting root asset: %v", err)
		}
		for _, floor := range locations {
			assetType := "abb_free_at_home_floor"
			_, floorAssetID, err := upsertAsset(assetData{
				config:                  config,
				projectId:               projectId,
				parentFunctionalAssetId: &rootAssetID,
				parentLocationalAssetId: &rootAssetID,
				identifier:              floor.GAI(),
				assetType:               assetType,
				name:                    floor.Name,
				description:             fmt.Sprintf("%s (%v)", floor.Name, floor.GAI()),
			})
			if err != nil {
				return fmt.Errorf("upserting floor %s: %v", floor.GAI(), err)
			}
			for _, room := range floor.Rooms {
				assetType := "abb_free_at_home_room"
				_, _, err := upsertAsset(assetData{
					config:                  config,
					projectId:               projectId,
					parentFunctionalAssetId: &floorAssetID,
					parentLocationalAssetId: &floorAssetID,
					identifier:              room.GAI(),
					assetType:               assetType,
					name:                    room.Name,
					description:             fmt.Sprintf("%s (%v)", room.Name, room.GAI()),
				})
				if err != nil {
					return fmt.Errorf("upserting room %s: %v", room.GAI(), err)
				}
			}
		}
	}
	return nil
}

func CreateAssetsIfNecessary(config apiserver.Configuration, systems []model.System) error {
	for _, projectId := range conf.ProjIds(config) {
		rootAssetID, err := upsertRootAsset(config, projectId)
		if err != nil {
			return fmt.Errorf("upserting root asset: %v", err)
		}
		for _, system := range systems {
			if len(system.Devices) == 0 {
				continue
			}
			assetType := "abb_free_at_home_system"
			_, systemAssetID, err := upsertAsset(assetData{
				config:                  config,
				projectId:               projectId,
				parentFunctionalAssetId: &rootAssetID,
				parentLocationalAssetId: &rootAssetID,
				identifier:              fmt.Sprintf("%s_%s", assetType, system.GAI),
				assetType:               assetType,
				name:                    system.Name,
				description:             fmt.Sprintf("%s (%v)", system.Name, system.GAI),
			})
			if err != nil {
				return fmt.Errorf("upserting system %s: %v", system.GAI, err)
			}
			for _, device := range system.Devices {
				if len(device.Channels) == 0 {
					continue
				}
				assetType := "abb_free_at_home_device"
				ad := assetData{
					config:                  config,
					projectId:               projectId,
					parentFunctionalAssetId: &systemAssetID,
					identifier:              fmt.Sprintf("%s_%s", assetType, device.GAI),
					assetType:               assetType,
					name:                    device.Name,
					description:             fmt.Sprintf("%s (%v)", device.Name, device.GAI),
				}

				locParentId := lookupLocationParent(config, projectId, device.Location)
				if locParentId == nil {
					locParentId = &systemAssetID
				}
				ad.parentLocationalAssetId = locParentId

				_, deviceAssetID, err := upsertAsset(ad)
				if err != nil {
					return fmt.Errorf("upserting device %s: %v", device.GAI, err)
				}
				for _, channel := range device.Channels {
					created, channelAssetID, err := upsertAsset(assetData{
						config:                  config,
						projectId:               projectId,
						parentFunctionalAssetId: &deviceAssetID,
						parentLocationalAssetId: &deviceAssetID,
						identifier:              channel.GAI(),
						assetType:               channel.AssetType(),
						name:                    channel.Name(),
						description:             fmt.Sprintf("%s (%v)", channel.Name(), channel.GAI()),
					})
					if err != nil {
						return fmt.Errorf("upserting channel %s: %v", channel.GAI(), err)
					}
					if created {
						for function, datapoint := range channel.Inputs() {
							_, err := conf.InsertInput(channelAssetID, system.ID, device.ID, channel.Id(), datapoint, function)
							if err != nil {
								return fmt.Errorf("inserting input: %v", err)
							}
						}
						for function, datapoint := range channel.Outputs() {
							dpId, err := conf.InsertOutput(channelAssetID, system.ID, device.ID, channel.Id(), datapoint.Name, function)
							if err != nil {
								return fmt.Errorf("inserting output: %v", err)
							}
							for _, attr := range datapoint.Map {
								if err := conf.LinkDatapointToAttribute(dpId, string(attr.Subtype), attr.AttributeName); err != nil {
									return fmt.Errorf("inserting datapoint-attribute link: %v", err)
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func lookupLocationParent(config apiserver.Configuration, projectId string, locationId string) *int32 {
	parentId, err := conf.GetAssetId(context.Background(), config, projectId, "abb_free_at_home_room_"+locationId)
	if err != nil {
		log.Debug("conf", "looking up asset location parent %v: %v", "abb_free_at_home_room_"+locationId, err)
		// Ignore. No location is a valid result as well.
		return nil
	}
	return parentId
}

func upsertRootAsset(config apiserver.Configuration, projectId string) (int32, error) {
	_, rootAssetID, err := upsertAsset(assetData{
		config:                  config,
		projectId:               projectId,
		parentLocationalAssetId: nil,
		identifier:              "abb_free_at_home_root",
		assetType:               "abb_free_at_home_root",
		name:                    "ABB-free@home",
		description:             "Root asset for ABB-free@home devices",
	})
	return rootAssetID, err
}

type assetData struct {
	config                  apiserver.Configuration
	projectId               string
	parentFunctionalAssetId *int32
	parentLocationalAssetId *int32
	identifier              string
	assetType               string
	name                    string
	description             string
}

func upsertAsset(d assetData) (created bool, assetID int32, err error) {
	// Get known asset id from configuration
	currentAssetID, err := conf.GetAssetId(context.Background(), d.config, d.projectId, d.identifier)
	if err != nil {
		return false, 0, fmt.Errorf("finding asset ID: %v", err)
	}
	if currentAssetID != nil {
		return false, *currentAssetID, nil
	}

	a := api.Asset{
		ProjectId:               d.projectId,
		GlobalAssetIdentifier:   d.identifier,
		Name:                    *api.NewNullableString(common.Ptr(d.name)),
		AssetType:               d.assetType,
		Description:             *api.NewNullableString(common.Ptr(d.description)),
		ParentFunctionalAssetId: *api.NewNullableInt32(d.parentFunctionalAssetId),
		ParentLocationalAssetId: *api.NewNullableInt32(d.parentLocationalAssetId),
		IsTracker:               *api.NewNullableBool(common.Ptr(false)),
	}
	newID, err := asset.UpsertAsset(a)
	if err != nil {
		return false, 0, fmt.Errorf("upserting asset %+v into Eliona: %v", a, err)
	}
	if newID == nil {
		return false, 0, fmt.Errorf("cannot create asset %s", d.name)
	}

	if err := conf.InsertAsset(context.Background(), d.config, d.projectId, d.identifier, d.assetType, *newID); err != nil {
		return false, 0, fmt.Errorf("inserting asset to config db: %v", err)
	}

	log.Debug("eliona", "Created new asset for project %s and device %s.", d.projectId, d.identifier)

	return true, *newID, nil
}
