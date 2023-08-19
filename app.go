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

package main

import (
	"abb-free-at-home/apiserver"
	"abb-free-at-home/apiservices"
	"abb-free-at-home/broker"
	"abb-free-at-home/conf"
	"abb-free-at-home/eliona"
	"context"
	"net/http"
	"time"

	"github.com/eliona-smart-building-assistant/go-utils/common"
	utilshttp "github.com/eliona-smart-building-assistant/go-utils/http"
	"github.com/eliona-smart-building-assistant/go-utils/log"
)

// collectData is the main app function which is called periodically
func collectData() {
	configs, err := conf.GetConfigs(context.Background())
	if err != nil {
		log.Fatal("conf", "Couldn't read configs from DB: %v", err)
		return
	}
	if len(configs) == 0 {
		log.Info("conf", "No configs in DB")
		return
	}

	for _, config := range configs {
		if !conf.IsConfigEnabled(config) {
			if conf.IsConfigActive(config) {
				conf.SetConfigActiveState(context.Background(), config, false)
			}
			continue
		}

		if !conf.IsConfigActive(config) {
			conf.SetConfigActiveState(context.Background(), config, true)
			log.Info("conf", "Collecting initialized with Configuration %d:\n"+
				"Enable: %t\n"+
				"Refresh Interval: %d\n"+
				"Request Timeout: %d\n"+
				"Project IDs: %v\n",
				*config.Id,
				*config.Enable,
				config.RefreshInterval,
				*config.RequestTimeout,
				*config.ProjectIDs)
		}

		common.RunOnceWithParam(func(config apiserver.Configuration) {
			log.Info("main", "Collecting %d started", *config.Id)

			if err := collectResources(config); err != nil {
				return // Error is handled in the method itself.
			}

			log.Info("main", "Collecting %d finished", *config.Id)

			time.Sleep(time.Second * time.Duration(config.RefreshInterval))
		}, config, *config.Id)
	}
}

func collectResources(config apiserver.Configuration) error {
	systems, err := broker.GetSystems(config)
	if err != nil {
		log.Error("abb", "getting abb configuration: %v", err)
		return err
	}
	if err := eliona.CreateAssetsIfNecessary(config, systems); err != nil {
		log.Error("eliona", "creating assets: %v", err)
		return err
	}

	// for _, v := range equipment {
	// 	assets = append(assets, v)
	// }

	// if err := eliona.UpsertAssetData(config, assets); err != nil {
	// 	log.Error("eliona", "inserting room data into Eliona: %v", err)
	// 	return err

	// }
	return nil
}

// listenApi starts the API server and listen for requests
func listenApi() {
	err := http.ListenAndServe(":"+common.Getenv("API_SERVER_PORT", "3000"), utilshttp.NewCORSEnabledHandler(
		apiserver.NewRouter(
			apiserver.NewConfigurationApiController(apiservices.NewConfigurationApiService()),
			apiserver.NewVersionApiController(apiservices.NewVersionApiService()),
			apiserver.NewCustomizationApiController(apiservices.NewCustomizationApiService()),
		)))
	log.Fatal("main", "API server: %v", err)
}
