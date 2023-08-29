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
	"abb-free-at-home/broker"
	"abb-free-at-home/conf"
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

func CreateAssetsIfNecessary(config apiserver.Configuration, systems []broker.System) error {
	for _, projectId := range conf.ProjIds(config) {
		rootAssetID, err := upsertRootAsset(config, projectId)
		if err != nil {
			return fmt.Errorf("upserting root asset: %v", err)
		}
		for _, system := range systems {
			assetType := "abb_free_at_home_system"
			_, systemAssetID, err := upsertAsset(assetData{
				config:                  config,
				projectId:               projectId,
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
				assetType := "abb_free_at_home_device"
				_, deviceAssetID, err := upsertAsset(assetData{
					config:                  config,
					projectId:               projectId,
					parentFunctionalAssetId: &systemAssetID,
					parentLocationalAssetId: &rootAssetID,
					identifier:              fmt.Sprintf("%s_%s", assetType, device.GAI),
					assetType:               assetType,
					name:                    device.Name,
					description:             fmt.Sprintf("%s (%v)", device.Name, device.GAI),
				})
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
						if sw, ok := channel.(broker.Switch); ok {
							for function, datapoint := range sw.Inputs {
								err := conf.InsertInput(channelAssetID, system.ID, device.ID, channel.Id(), datapoint, function)
								if err != nil {
									return fmt.Errorf("inserting input: %v", err)
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

	// Remember the asset id for further usage
	if err := conf.InsertAsset(context.Background(), d.config, d.projectId, d.identifier, *newID); err != nil {
		return false, 0, fmt.Errorf("inserting asset to config db: %v", err)
	}

	log.Debug("eliona", "Created new asset for project %s and device %s.", d.projectId, d.identifier)

	return true, *newID, nil
}
