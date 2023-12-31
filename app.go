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

var once sync.Once
var resynchronizeTrigger = make(chan struct{}, 1)

func collectData() {
	configs, err := conf.GetConfigs(context.Background())
	if err != nil {
		log.Fatal("conf", "Couldn't read configs from DB: %v", err)
		return
	}
	if len(configs) == 0 {
		once.Do(func() {
			log.Info("conf", "No configs in DB. Please configure the app in Eliona.")
		})
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
		collectAndStartSubscription(config)
	}
}

func collectAndStartSubscription(config apiserver.Configuration) {
	common.RunOnceWithParam(func(config apiserver.Configuration) {
		log.Info("main", "Collecting %d started", *config.Id)

		if err := collectResources(&config); err != nil {
			return // Error is handled in the method itself.
		}

		log.Info("main", "Collecting %d finished", *config.Id)

		common.RunOnceWithParam(func(config apiserver.Configuration) {
			log.Info("main", "Subscription %d started.", *config.Id)
			subscribeToDataChanges(&config)
			log.Info("main", "Subscription %d exited. Restarting ...", *config.Id)
			triggerResynchronize()
		}, config, fmt.Sprintf("subscription_%v", *config.Id))
		common.RunOnceWithParam(func(config apiserver.Configuration) {
			log.Info("main", "Status subscription %d started.", *config.Id)
			subscribeToSystemStatus(&config)
			log.Info("main", "Status subscription %d exited. Restarting ...", *config.Id)
			triggerResynchronize()
		}, config, fmt.Sprintf("status_subscription_%v", *config.Id))
		for {
			// Wait for the time duration or a trigger
			select {
			case <-time.After(time.Second * time.Duration(config.RefreshInterval)):
				return
			case <-resynchronizeTrigger:
				log.Info("main", "Resynchronization trigerred.")
				return
			}
		}
	}, config, *config.Id)
}

func triggerResynchronize() {
	// Non-blocking Send: This ensures that sending to the channel doesn't block if the channel buffer is full.
	select {
	case resynchronizeTrigger <- struct{}{}:
	default:
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

// ABB -> Eliona
func subscribeToDataChanges(config *apiserver.Configuration) {
	datapoints, err := conf.FetchAllDatapoints()
	if err != nil {
		log.Error("conf", "fetching all datapoints: %v", err)
		return
	}

	dataPointChan := make(chan abbgraphql.DataPoint)
	go func() {
		defer close(dataPointChan)

		if err := broker.ListenForDataChanges(config, datapoints, dataPointChan); err != nil {
			log.Error("broker", "listen for data changes: %v", err)
			return
		}
		log.Info("broker", "ABB subscription exited")
	}()
	for dp := range dataPointChan {
		datapoint, err := conf.FindDatapoint(dp.SerialNumber, dp.ChannelNumber, dp.DatapointId)
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

func subscribeToSystemStatus(config *apiserver.Configuration) {
	systems, err := conf.GetSystems(context.Background(), *config)
	if err != nil {
		log.Error("conf", "fetching all systems: %v", err)
		return
	}

	var dtIDs []string
	for _, s := range systems {
		dtIDs = append(dtIDs, s.ProviderID)
	}

	connectionStatusChan := make(chan abbgraphql.ConnectionStatus)
	go func() {
		defer close(connectionStatusChan)

		if err := broker.ListenForSystemStatusChanges(config, dtIDs, connectionStatusChan); err != nil {
			log.Error("broker", "listen for system status changes: %v", err)
			return
		}
		log.Info("broker", "ABB subscription exited")
	}()
	for status := range connectionStatusChan {
		log.Debug("broker", "status received: %v", status)
		system, err := conf.FindAssetByProviderID(context.Background(), *config, status.DtId)
		if err != nil {
			log.Error("conf", "finding system %+v: %v", status.DtId, err)
			return
		}
		connected := int8(0)
		if status.Connected {
			connected = 1
		}
		if err := eliona.UpsertSystemStatus(*config, *system, connected); err != nil {
			log.Error("eliona", "upserting system data %+v: %v", status.Connected, err)
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

// Eliona -> ABB
func listenForOutputChanges() {
	for { // We want to restart listening in case something breaks.
		outputs, err := eliona.ListenForOutputChanges()
		if err != nil {
			log.Error("eliona", "listening for output changes: %v", err)
			return
		}
		log.Debug("eliona", "started websocket listener")
		for output := range outputs {
			if cr := output.ClientReference.Get(); cr != nil && *cr == eliona.ClientReference {
				// Just an echoed value this app sent.
				continue
			}
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
					log.Error("app", "output: got value of unknown type: %v", val)
					continue
				}
				setAsset(output.AssetId, function, value)
			}
		}
		log.Warn("Eliona", "Websocket connection broke. Restarting in 5 seconds.")
		time.Sleep(time.Second * 5) // Give the server a little break.
	}
}

var assetWrites = make(map[int32]time.Time)

func setAsset(assetID int32, function string, val float64) {
	input, err := conf.FetchInput(assetID, function)
	if err != nil {
		log.Fatal("conf", "fetching input for assetID %v function %v: %v", assetID, function, err)
		return
	}

	if lastAssetWrite, ok := assetWrites[assetID]; ok {
		if time.Since(lastAssetWrite).Seconds() < 1 {
			log.Info("broker", "skipped setting value %v for function %v asset %v, to avoid overwriting dependent values", val, function, assetID)
			return
		} else {
			fmt.Println(time.Since(lastAssetWrite).Seconds())
		}
	}

	config, err := conf.GetConfigForDatapoint(input)
	if err != nil {
		log.Error("conf", "getting config for input %v: %v", input.ID, err)
		return
	}
	log.Info("broker", "setting value %v for asset %v function %v", val, assetID, function)
	if err := broker.SetInput(&config, input, val); err != nil {
		log.Error("broker", "setting value for asset %v: %v", assetID, err)
		return
	}
	assetWrites[assetID] = time.Now()
	input.LastWrittenValue.Float64 = val
	input.LastWrittenValue.Valid = true
	input.LastWrittenTime.Time = time.Now()
	input.LastWrittenTime.Valid = true
	if err := conf.UpdateDatapoint(input); err != nil {
		log.Error("conf", "updating input: %v", err)
		return
	}

	// This is an ugly hack to handle state when the RTC is in ECO mode. First
	// call turns off the eco mode (and highens the temperature), second call
	// really sets the temperature.
	if function == broker.SET_TEMP_TWICE {
		if err := conf.UpdateDatapoint(input); err != nil {
			log.Error("conf", "updating input second time: %v", err)
			return
		}
	}
}
