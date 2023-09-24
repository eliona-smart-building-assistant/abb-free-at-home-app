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
	"abb-free-at-home/abbgraphql"
	"abb-free-at-home/apiserver"
	"abb-free-at-home/apiservices"
	"abb-free-at-home/broker"
	"abb-free-at-home/conf"
	"abb-free-at-home/eliona"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/eliona-smart-building-assistant/go-utils/common"
	utilshttp "github.com/eliona-smart-building-assistant/go-utils/http"
	"github.com/eliona-smart-building-assistant/go-utils/log"
)

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

			if err := collectResources(&config); err != nil {
				return // Error is handled in the method itself.
			}

			log.Info("main", "Collecting %d finished", *config.Id)

			common.RunOnceWithParam(func(config apiserver.Configuration) {
				log.Info("main", "subscription %d started", *config.Id)
				subscribeToDataChanges(&config)
				log.Info("main", "subscription %d finished", *config.Id)
			}, config, fmt.Sprintf("subscription_%v", *config.Id))
			time.Sleep(time.Second * time.Duration(config.RefreshInterval))
		}, config, *config.Id)
	}
}

func collectResources(config *apiserver.Configuration) error {
	locations, err := broker.GetLocations(config)
	if err != nil {
		log.Error("abb", "getting abb locations: %v", err)
		return err
	}
	if err := eliona.CreateLocationAssetsIfNecessary(*config, locations); err != nil {
		log.Error("eliona", "creating location assets: %v", err)
		return err
	}

	systems, err := broker.GetSystems(config)
	if err != nil {
		log.Error("abb", "getting abb configuration: %v", err)
		return err
	}
	if err := eliona.CreateAssetsIfNecessary(*config, systems); err != nil {
		log.Error("eliona", "creating assets: %v", err)
		return err
	}

	if err := eliona.UpsertSystemsData(*config, systems); err != nil {
		log.Error("eliona", "inserting data into Eliona: %v", err)
		return err
	}
	return nil
}

func subscribeToDataChanges(config *apiserver.Configuration) {
	datapoints, err := conf.FetchAllDatapoints()
	if err != nil {
		log.Error("conf", "fetching all datapoints: %v", err)
		return
	}

	dataPointChan := make(chan abbgraphql.DataPoint)
	go func() {
		if err := broker.ListenForDataChanges(config, datapoints, dataPointChan); err != nil {
			log.Error("broker", "listen for data changes: %v", err)
			return
		}
	}()
	for dp := range dataPointChan {
		if abbTimerActive() {
			return
		}
		startElionaTimer()
		datapoint, err := conf.FindDatapoint(string(dp.SerialNumber), string(dp.ChannelNumber), string(dp.DatapointId))
		if err != nil {
			log.Error("conf", "finding datapoint %+v: %v", dp, err)
			return
		}
		if err := eliona.UpsertDatapointData(*config, datapoint, dp.Value); err != nil {
			log.Error("eliona", "upserting datapoint data %+v: %v", dp, err)
			return
		}
	}
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

func listenForOutputChanges() {
	outputs, err := eliona.ListenForOutputChanges()
	if err != nil {
		log.Error("eliona", "listening for output changes: %v", err)
		return
	}
	for output := range outputs {
		for _, function := range broker.Functions {
			val, ok := output.Data[function]
			if !ok {
				continue
			}
			var value float64

			switch v := val.(type) {
			case float64:
				value = v
			case string:
				if value, err = strconv.ParseFloat(v, 64); err != nil {
					log.Error("app", "output: parsing %v: %v", v, err)
					continue
				}
			default:
				log.Error("app", "output: got non-float64 value %v", val)
				continue
			}
			setAsset(output.AssetId, function, value)
		}
	}
}

func setAsset(assetID int32, function string, val float64) {
	if elionaTimerActive() {
		return
	}
	startAbbTimer()
	input, err := conf.FetchInput(assetID, function)
	if err != nil {
		log.Fatal("conf", "fetching input for assetID %v function %v: %v", assetID, function, err)
		return
	}
	if input.LastWrittenValue.Valid && input.LastWrittenValue.Float64 == val {
		log.Info("broker", "skipped setting value %v for asset %v, same as last written", val, assetID)
		return
	}

	if lastAssetWrite, err := conf.LastWriteToAsset(assetID); err != nil {
		log.Error("conf", "fetching last asset write: %v", err)
		return
	} else if time.Since(lastAssetWrite).Seconds() < 3 {
		log.Debug("broker", "skipped setting value %v for asset %v, to avoid overwriting dependent values", val, assetID)
		return
	}

	config, err := conf.GetConfigForDatapoint(input)
	if err != nil {
		log.Error("conf", "getting config for input %v: %v", input.ID, err)
		return
	}
	log.Info("broker", "setting value %v for asset %v", val, assetID)
	if err := broker.SetInput(&config, input, val); err != nil {
		log.Error("broker", "setting value %v for asset %v: %v", val, assetID, err)
		return
	}
	input.LastWrittenValue.Float64 = val
	input.LastWrittenValue.Valid = true
	input.LastWrittenTime.Time = time.Now()
	input.LastWrittenTime.Valid = true
	if err := conf.UpdateDatapoint(input); err != nil {
		log.Error("conf", "updating input: %v", err)
		return
	}
}

var (
	mu                    sync.Mutex
	abbIgnoreTimer        *time.Timer
	elionaIgnoreTimer     *time.Timer
	abbTimerActiveFlag    bool
	elionaTimerActiveFlag bool
	ignoreDuration        = 500 * time.Millisecond
)

func startAbbTimer() {
	mu.Lock()
	defer mu.Unlock()

	if abbIgnoreTimer != nil {
		abbIgnoreTimer.Stop()
	}
	abbIgnoreTimer = time.NewTimer(ignoreDuration)
	abbTimerActiveFlag = true

	go func() {
		<-abbIgnoreTimer.C
		mu.Lock()
		abbTimerActiveFlag = false
		mu.Unlock()
	}()
}

func startElionaTimer() {
	mu.Lock()
	defer mu.Unlock()

	if elionaIgnoreTimer != nil {
		elionaIgnoreTimer.Stop()
	}
	elionaIgnoreTimer = time.NewTimer(ignoreDuration)
	elionaTimerActiveFlag = true

	go func() {
		<-elionaIgnoreTimer.C
		mu.Lock()
		elionaTimerActiveFlag = false
		mu.Unlock()
	}()
}

func abbTimerActive() bool {
	mu.Lock()
	defer mu.Unlock()
	return abbTimerActiveFlag
}

func elionaTimerActive() bool {
	mu.Lock()
	defer mu.Unlock()
	return elionaTimerActiveFlag
}
