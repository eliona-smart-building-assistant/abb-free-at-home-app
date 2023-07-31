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

package apiservices

import (
	"abb-free-at-home/apiserver"
	"abb-free-at-home/conf"
	"context"
	"errors"
	"net/http"
)

// ConfigurationApiService is a service that implements the logic for the ConfigurationApiServicer
// This service should implement the business logic for every endpoint for the ConfigurationApi API.
// Include any external packages or services that will be required by this service.
type ConfigurationApiService struct {
}

// NewConfigurationApiService creates a default api service
func NewConfigurationApiService() apiserver.ConfigurationApiServicer {
	return &ConfigurationApiService{}
}

func (s *ConfigurationApiService) GetConfigurations(ctx context.Context) (apiserver.ImplResponse, error) {
	configs, err := conf.GetConfigs(ctx)
	if err != nil {
		return apiserver.ImplResponse{Code: http.StatusInternalServerError}, err
	}
	return apiserver.Response(http.StatusOK, configs), nil
}

func (s *ConfigurationApiService) PostConfiguration(ctx context.Context, config apiserver.Configuration) (apiserver.ImplResponse, error) {
	insertedConfig, err := conf.InsertConfig(ctx, config)
	if err != nil {
		return apiserver.ImplResponse{Code: http.StatusInternalServerError}, err
	}
	return apiserver.Response(http.StatusCreated, insertedConfig), nil
}

func (s *ConfigurationApiService) GetConfigurationById(ctx context.Context, configId int64) (apiserver.ImplResponse, error) {
	config, err := conf.GetConfig(ctx, configId)
	if errors.Is(err, conf.ErrBadRequest) {
		return apiserver.ImplResponse{Code: http.StatusBadRequest}, nil
	}
	if err != nil {
		return apiserver.ImplResponse{Code: http.StatusInternalServerError}, err
	}
	return apiserver.Response(http.StatusOK, config), nil
}

func (s *ConfigurationApiService) PutConfigurationById(ctx context.Context, configId int64, config apiserver.Configuration) (apiserver.ImplResponse, error) {
	config.Id = &configId
	upsertedConfig, err := conf.InsertConfig(ctx, config)
	if err != nil {
		return apiserver.ImplResponse{Code: http.StatusInternalServerError}, err
	}
	return apiserver.Response(http.StatusCreated, upsertedConfig), nil
}

func (s *ConfigurationApiService) DeleteConfigurationById(ctx context.Context, configId int64) (apiserver.ImplResponse, error) {
	err := conf.DeleteConfig(ctx, configId)
	if errors.Is(err, conf.ErrBadRequest) {
		return apiserver.ImplResponse{Code: http.StatusBadRequest}, nil
	}
	if err != nil {
		return apiserver.ImplResponse{Code: http.StatusInternalServerError}, err
	}
	return apiserver.ImplResponse{Code: http.StatusNoContent}, nil
}
