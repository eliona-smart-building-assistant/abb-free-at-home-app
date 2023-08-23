package eliona

import (
	"abb-free-at-home/apiserver"
	"abb-free-at-home/broker"
	"abb-free-at-home/conf"
	"context"
	"fmt"

	"github.com/eliona-smart-building-assistant/go-eliona/asset"
	"github.com/eliona-smart-building-assistant/go-utils/log"
)

func UpsertSystemsData(config apiserver.Configuration, systems []broker.System) error {
	for _, projectId := range *config.ProjectIDs {
		for _, system := range systems {
			for _, device := range system.Devices {
				for _, channel := range device.Channels {
					log.Debug("Eliona", "upserting data for asset: config %d and asset '%v'", config.Id, channel.Id())
					assetId, err := conf.GetAssetId(context.Background(), config, projectId, channel.Id())
					if err != nil {
						return err
					}
					if assetId == nil {
						return fmt.Errorf("unable to find asset ID")
					}

					data := asset.Data{
						AssetId: *assetId,
						Data:    channel,
					}
					if asset.UpsertAssetDataIfAssetExists(data); err != nil {
						return fmt.Errorf("upserting data: %v", err)
					}
				}
			}
		}
	}
	return nil
}
