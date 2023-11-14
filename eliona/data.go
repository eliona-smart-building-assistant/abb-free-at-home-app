package eliona

import (
	"abb-free-at-home/apiserver"
	"abb-free-at-home/appdb"
	"abb-free-at-home/conf"
	"abb-free-at-home/model"
	"context"
	"fmt"

	api "github.com/eliona-smart-building-assistant/go-eliona-api-client/v2"
	"github.com/eliona-smart-building-assistant/go-eliona/asset"
	"github.com/eliona-smart-building-assistant/go-utils/log"
)

const ClientReference string = "abb-free-at-home"

func UpsertSystemsData(config apiserver.Configuration, systems []model.System) error {
	for _, projectId := range *config.ProjectIDs {
		for _, system := range systems {
			for _, device := range system.Devices {
				for _, channel := range device.Channels {
					log.Debug("Eliona", "upserting data for asset: config %d and asset '%v'", config.Id, channel.GAI())
					assetId, err := conf.GetAssetId(context.Background(), config, projectId, channel.GAI())
					if err != nil {
						return err
					}
					if assetId == nil {
						return fmt.Errorf("unable to find asset ID")
					}

					data := asset.Data{
						AssetId:         *assetId,
						Data:            channel,
						ClientReference: ClientReference,
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

func UpsertDatapointData(config apiserver.Configuration, datapoint appdb.Datapoint, value any) error {
	attributes, err := datapoint.DatapointAttributes().AllG(context.Background())
	if err != nil {
		return fmt.Errorf("fetching datapoint attributes: %v", err)
	}
	ast, err := datapoint.Asset().OneG(context.Background())
	if err != nil {
		return fmt.Errorf("fetching datapoint asset: %v", err)
	}
	for _, projectId := range *config.ProjectIDs {
		for _, attribute := range attributes {
			log.Debug("Eliona", "upserting data for asset: config %d and asset '%v'", config.Id, ast.GlobalAssetID)
			assetId, err := conf.GetAssetId(context.Background(), config, projectId, ast.GlobalAssetID)
			if err != nil {
				return err
			}
			if assetId == nil {
				return fmt.Errorf("unable to find asset ID")
			}
			data := map[string]interface{}{
				attribute.AttributeName: value,
			}

			cr := ClientReference
			apidata := api.Data{
				AssetId:         *assetId,
				Data:            data,
				Subtype:         api.DataSubtype(attribute.Subtype),
				AssetTypeName:   *api.NewNullableString(&ast.AssetTypeName),
				ClientReference: *api.NewNullableString(&cr),
			}
			if asset.UpsertDataIfAssetExists(apidata); err != nil {
				return fmt.Errorf("upserting data: %v", err)
			}
		}
	}
	return nil
}
