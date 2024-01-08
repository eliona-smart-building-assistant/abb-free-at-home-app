package eliona

import (
	"abb-free-at-home/apiserver"
	"abb-free-at-home/appdb"
	"abb-free-at-home/conf"
	"abb-free-at-home/model"
	"context"
	"fmt"
	"strconv"
	"strings"

	api "github.com/eliona-smart-building-assistant/go-eliona-api-client/v2"
	"github.com/eliona-smart-building-assistant/go-eliona/asset"
	"github.com/eliona-smart-building-assistant/go-utils/log"
)

const ClientReference string = "abb-free-at-home"

func UpsertSystemsData(config apiserver.Configuration, systems []model.System) error {
	for _, projectId := range *config.ProjectIDs {
		for _, system := range systems {
			log.Debug("Eliona", "upserting data for system: config %d and system '%s'", config.Id, fmt.Sprintf("%s_%s", system.AssetType(), system.GAI))
			assetId, err := conf.GetAssetId(context.Background(), config, projectId, fmt.Sprintf("%s_%s", system.AssetType(), system.GAI))
			if err != nil {
				return err
			}
			if assetId == nil {
				continue
			}

			data := asset.Data{
				AssetId:         *assetId,
				Data:            system,
				ClientReference: ClientReference,
			}
			if asset.UpsertAssetDataIfAssetExists(data); err != nil {
				return fmt.Errorf("upserting data: %v", err)
			}
			for _, device := range system.Devices {
				log.Debug("Eliona", "upserting data for device: config %d and device '%s'", config.Id, fmt.Sprintf("%s_%s", device.AssetType(), device.GAI))
				assetId, err := conf.GetAssetId(context.Background(), config, projectId, fmt.Sprintf("%s_%s", device.AssetType(), device.GAI))
				if err != nil {
					return err
				}
				if assetId == nil {
					continue
				}

				data := asset.Data{
					AssetId:         *assetId,
					Data:            device,
					ClientReference: ClientReference,
				}
				if asset.UpsertAssetDataIfAssetExists(data); err != nil {
					return fmt.Errorf("upserting data: %v", err)
				}
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

func UpsertDatapointData(config apiserver.Configuration, datapoint appdb.Datapoint, value string) error {
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
			log.Debug("Eliona", "upserting data for datapoint: config %d and asset '%v'", config.Id, ast.GlobalAssetID)
			assetId, err := conf.GetAssetId(context.Background(), config, projectId, ast.GlobalAssetID)
			if err != nil {
				return err
			}
			if assetId == nil {
				return fmt.Errorf("unable to find asset ID")
			}
			data := map[string]interface{}{
				attribute.AttributeName: convertToNumber(value),
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

func UpsertSystemStatus(config apiserver.Configuration, system appdb.Asset, status int8) error {
	for _, projectId := range *config.ProjectIDs {
		log.Debug("Eliona", "upserting status for system: config %d and asset '%v'", config.Id, system.GlobalAssetID)
		assetId, err := conf.GetAssetId(context.Background(), config, projectId, system.GlobalAssetID)
		if err != nil {
			return err
		}
		if assetId == nil {
			return fmt.Errorf("unable to find asset ID")
		}
		data := map[string]interface{}{
			"connection_status": status,
		}

		cr := ClientReference
		apidata := api.Data{
			AssetId:         *assetId,
			Data:            data,
			Subtype:         api.DataSubtype(api.SUBTYPE_STATUS),
			AssetTypeName:   *api.NewNullableString(&system.AssetTypeName),
			ClientReference: *api.NewNullableString(&cr),
		}
		if asset.UpsertDataIfAssetExists(apidata); err != nil {
			return fmt.Errorf("upserting data: %v", err)
		}
	}
	return nil
}

// convertToNumber tries to convert a string to an integer or a float.
// If conversion is not possible, it returns the original string.
func convertToNumber(s string) any {
	if strings.Contains(s, ".") {
		// Try converting to float
		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return s
		}
		return val
	}
	// Try converting to integer
	val, err := strconv.Atoi(s)
	if err != nil {
		return s
	}
	return val
}
