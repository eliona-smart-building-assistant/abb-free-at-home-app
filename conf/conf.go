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

package conf

import (
	"abb-free-at-home/apiserver"
	"abb-free-at-home/appdb"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/eliona-smart-building-assistant/go-utils/common"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"golang.org/x/oauth2"
)

var ErrBadRequest = errors.New("bad request")

func InsertConfig(ctx context.Context, config apiserver.Configuration) (apiserver.Configuration, error) {
	dbConfig, err := dbConfigFromApiConfig(config)
	if err != nil {
		return apiserver.Configuration{}, fmt.Errorf("creating DB config from API config: %v", err)
	}
	if err := dbConfig.InsertG(ctx, boil.Infer()); err != nil {
		return apiserver.Configuration{}, fmt.Errorf("inserting DB config: %v", err)
	}
	return config, nil
}

func GetConfig(ctx context.Context, configID int64) (*apiserver.Configuration, error) {
	dbConfig, err := appdb.Configurations(
		appdb.ConfigurationWhere.ID.EQ(configID),
	).OneG(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching config from database")
	}
	if dbConfig == nil {
		return nil, ErrBadRequest
	}
	apiConfig, err := apiConfigFromDbConfig(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("creating API config from DB config: %v", err)
	}
	return &apiConfig, nil
}

func DeleteConfig(ctx context.Context, configID int64) error {
	count, err := appdb.Configurations(
		appdb.ConfigurationWhere.ID.EQ(configID),
	).DeleteAllG(ctx)
	if err != nil {
		return fmt.Errorf("fetching config from database")
	}
	if count > 1 {
		return fmt.Errorf("shouldn't happen: deleted more (%v) configs by ID", count)
	}
	if count == 0 {
		return ErrBadRequest
	}
	return nil
}

func dbConfigFromApiConfig(apiConfig apiserver.Configuration) (dbConfig appdb.Configuration, err error) {
	dbConfig.IsCloud = apiConfig.IsCloud
	if apiConfig.ClientID != nil {
		dbConfig.ClientID.String = *apiConfig.ClientID
		dbConfig.ClientID.Valid = true
	}
	if apiConfig.ClientSecret != nil {
		dbConfig.ClientSecret.String = *apiConfig.ClientSecret
		dbConfig.ClientSecret.Valid = true
	}
	if apiConfig.AccessToken != nil {
		dbConfig.AccessToken.String = *apiConfig.AccessToken
		dbConfig.AccessToken.Valid = true
	}
	if apiConfig.RefreshToken != nil {
		dbConfig.RefreshToken.String = *apiConfig.RefreshToken
		dbConfig.RefreshToken.Valid = true
	}
	if apiConfig.Expiry != nil {
		dbConfig.Expiry.Time = *apiConfig.Expiry
		dbConfig.Expiry.Valid = true
	}
	if apiConfig.ApiUrl != nil {
		dbConfig.APIURL.String = *apiConfig.ApiUrl
		dbConfig.APIURL.Valid = true
	}
	if apiConfig.ApiUsername != nil {
		dbConfig.APIUsername.String = *apiConfig.ApiUsername
		dbConfig.APIUsername.Valid = true
	}
	if apiConfig.ApiPassword != nil {
		dbConfig.APIPassword.String = *apiConfig.ApiPassword
		dbConfig.APIPassword.Valid = true
	}

	dbConfig.ID = null.Int64FromPtr(apiConfig.Id).Int64
	dbConfig.Enable = null.BoolFromPtr(apiConfig.Enable)
	dbConfig.RefreshInterval = apiConfig.RefreshInterval
	if apiConfig.RequestTimeout != nil {
		dbConfig.RequestTimeout = *apiConfig.RequestTimeout
	}
	af, err := json.Marshal(apiConfig.AssetFilter)
	if err != nil {
		return appdb.Configuration{}, fmt.Errorf("marshalling assetFilter: %v", err)
	}
	dbConfig.AssetFilter = null.JSONFrom(af)
	dbConfig.Active = null.BoolFromPtr(apiConfig.Active)
	if apiConfig.ProjectIDs != nil {
		dbConfig.ProjectIds = *apiConfig.ProjectIDs
	}

	return dbConfig, nil
}

func apiConfigFromDbConfig(dbConfig *appdb.Configuration) (apiConfig apiserver.Configuration, err error) {
	apiConfig.IsCloud = dbConfig.IsCloud
	apiConfig.ClientID = dbConfig.ClientID.Ptr()
	apiConfig.ClientSecret = dbConfig.ClientSecret.Ptr()
	apiConfig.AccessToken = dbConfig.AccessToken.Ptr()
	apiConfig.RefreshToken = dbConfig.RefreshToken.Ptr()
	if dbConfig.Expiry.Valid {
		apiConfig.Expiry = &dbConfig.Expiry.Time
	}
	apiConfig.ApiUrl = dbConfig.APIURL.Ptr()
	apiConfig.ApiUsername = dbConfig.APIUsername.Ptr()
	apiConfig.ApiPassword = dbConfig.APIPassword.Ptr()

	apiConfig.Id = &dbConfig.ID
	apiConfig.Enable = dbConfig.Enable.Ptr()
	apiConfig.RefreshInterval = dbConfig.RefreshInterval
	apiConfig.RequestTimeout = &dbConfig.RequestTimeout
	if dbConfig.AssetFilter.Valid {
		var af [][]apiserver.FilterRule
		if err := json.Unmarshal(dbConfig.AssetFilter.JSON, &af); err != nil {
			return apiserver.Configuration{}, fmt.Errorf("unmarshalling assetFilter: %v", err)
		}
		apiConfig.AssetFilter = af
	}
	apiConfig.Active = dbConfig.Active.Ptr()
	apiConfig.ProjectIDs = common.Ptr[[]string](dbConfig.ProjectIds)
	return apiConfig, nil
}

func GetConfigs(ctx context.Context) ([]apiserver.Configuration, error) {
	dbConfigs, err := appdb.Configurations().AllG(ctx)
	if err != nil {
		return nil, err
	}
	var apiConfigs []apiserver.Configuration
	for _, dbConfig := range dbConfigs {
		ac, err := apiConfigFromDbConfig(dbConfig)
		if err != nil {
			return nil, fmt.Errorf("creating API config from DB config: %v", err)
		}
		apiConfigs = append(apiConfigs, ac)
	}
	return apiConfigs, nil
}

func SetConfigActiveState(ctx context.Context, config apiserver.Configuration, state bool) (int64, error) {
	return appdb.Configurations(
		appdb.ConfigurationWhere.ID.EQ(null.Int64FromPtr(config.Id).Int64),
	).UpdateAllG(ctx, appdb.M{
		appdb.ConfigurationColumns.Active: state,
	})
}

func ProjIds(config apiserver.Configuration) []string {
	if config.ProjectIDs == nil {
		return []string{}
	}
	return *config.ProjectIDs
}

func IsConfigActive(config apiserver.Configuration) bool {
	return config.Active == nil || *config.Active
}

func IsConfigEnabled(config apiserver.Configuration) bool {
	return config.Enable == nil || *config.Enable
}

func SetAllConfigsInactive(ctx context.Context) (int64, error) {
	return appdb.Configurations().UpdateAllG(ctx, appdb.M{
		appdb.ConfigurationColumns.Active: false,
	})
}

func PersistAuthorization(config apiserver.Configuration, auth oauth2.Token) (int64, error) {
	return appdb.Configurations(
		appdb.ConfigurationWhere.ID.EQ(*config.Id),
	).UpdateAllG(context.Background(), appdb.M{
		appdb.ConfigurationColumns.AccessToken:  auth.AccessToken,
		appdb.ConfigurationColumns.RefreshToken: auth.RefreshToken,
		appdb.ConfigurationColumns.Expiry:       auth.Expiry,
	})
}

func InvalidateAuthorization(config apiserver.Configuration) (int64, error) {
	return appdb.Configurations(
		appdb.ConfigurationWhere.ID.EQ(*config.Id),
	).UpdateAllG(context.Background(), appdb.M{
		appdb.ConfigurationColumns.AccessToken:  nil,
		appdb.ConfigurationColumns.RefreshToken: nil,
		appdb.ConfigurationColumns.Expiry:       nil,
	})
}

func InsertAsset(ctx context.Context, config apiserver.Configuration, projId string, globalAssetID string, assetId int32) error {
	var dbAsset appdb.Asset
	dbAsset.ConfigurationID = null.Int64FromPtr(config.Id).Int64
	dbAsset.ProjectID = projId
	dbAsset.GlobalAssetID = globalAssetID
	dbAsset.AssetID = null.Int32From(assetId)
	return dbAsset.InsertG(ctx, boil.Infer())
}

func GetAssetId(ctx context.Context, config apiserver.Configuration, projId string, globalAssetID string) (*int32, error) {
	dbAsset, err := appdb.Assets(
		appdb.AssetWhere.ConfigurationID.EQ(null.Int64FromPtr(config.Id).Int64),
		appdb.AssetWhere.ProjectID.EQ(projId),
		appdb.AssetWhere.GlobalAssetID.EQ(globalAssetID),
	).AllG(ctx)
	if err != nil || len(dbAsset) == 0 {
		return nil, err
	}
	return common.Ptr(dbAsset[0].AssetID.Int32), nil
}

func InsertInput(assetId int32, systemId, deviceId, channelId, datapoint, function string) error {
	input := appdb.Input{
		AssetID:   assetId,
		SystemID:  systemId,
		DeviceID:  deviceId,
		ChannelID: channelId,
		Datapoint: datapoint,
		Function:  function,
	}
	return input.InsertG(context.Background(), boil.Infer())
}

func FetchInput(assetId int32, function string) (appdb.Input, error) {
	input, err := appdb.Inputs(
		appdb.InputWhere.AssetID.EQ(assetId),
		appdb.InputWhere.Function.EQ(function),
	).OneG(context.Background())
	if err != nil {
		return appdb.Input{}, err
	}
	return *input, nil
}

func UpdateInput(input appdb.Input) error {
	_, err := input.UpdateG(context.Background(), boil.Infer())
	return err
}

func GetConfigForInput(input appdb.Input) (config apiserver.Configuration, err error) {
	asset, err := input.Asset().OneG(context.Background())
	if err != nil {
		err = fmt.Errorf("fetching asset: %v", err)
		return
	}
	c, err := asset.Configuration().OneG(context.Background())
	if err != nil {
		err = fmt.Errorf("fetching configuration: %v", err)
		return
	}
	return apiConfigFromDbConfig(c)
}
