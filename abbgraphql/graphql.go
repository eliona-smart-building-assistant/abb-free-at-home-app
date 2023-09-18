package abbgraphql

import (
	"context"
	"fmt"
	"net/http"

	graphql "github.com/hasura/go-graphql-client"
)

type SystemsQuery struct {
	Refresh struct {
		Refreshed graphql.Boolean `graphql:"refreshed"`
	} `graphql:"Refresh"`
	Systems []struct {
		DtId   graphql.String `graphql:"dtId"`
		Assets []struct {
			SerialNumber graphql.String `graphql:"serialNumber"`
			Name         struct {
				En graphql.String `graphql:"en"`
			} `graphql:"Name"`
			Channels []struct {
				ChannelNumber graphql.Int    `graphql:"channelNumber"`
				FunctionId    graphql.String `graphql:"functionId"`
				Name          struct {
					En graphql.String `graphql:"en"`
				} `graphql:"Name"`
				Outputs []struct {
					Key   graphql.String `graphql:"key"`
					Value struct {
						PairingId graphql.String `graphql:"pairingId"`
						Name      struct {
							En graphql.String `graphql:"en"`
						} `graphql:"Name"`
						Dpt              graphql.String `graphql:"dpt"`
						DataPointService struct {
							RequestDataPointValue struct {
								Value graphql.String `graphql:"value"`
								Time  graphql.String `graphql:"time"`
							} `graphql:"RequestDataPointValue"`
						} `graphql:"DataPointService"`
					} `graphql:"value"`
				} `graphql:"outputs"`
				Inputs []struct {
					Key   graphql.String `graphql:"key"`
					Value struct {
						PairingId graphql.String `graphql:"pairingId"`
						Name      struct {
							En graphql.String `graphql:"en"`
						} `graphql:"Name"`
						Dpt              graphql.String `graphql:"dpt"`
						DataPointService struct {
							RequestDataPointValue struct {
								Value graphql.String `graphql:"value"`
								Time  graphql.String `graphql:"time"`
							} `graphql:"RequestDataPointValue"`
						} `graphql:"DataPointService"`
					} `graphql:"value"`
				} `graphql:"inputs"`
			} `graphql:"Channels"`
		} `graphql:"Assets"`
	} `graphql:"ISystemFH"`
}

func GetSystems(httpClient *http.Client) (SystemsQuery, error) {
	client := getClient(httpClient)
	var query SystemsQuery
	variables := map[string]interface{}{}
	if err := client.Query(context.Background(), &query, variables); err != nil {
		return SystemsQuery{}, err
	}
	return query, nil
}

type setQuery struct {
	IDeviceFH []struct {
		Channels []struct {
			ChannelNumber graphql.Int `graphql:"channelNumber"`
			Inputs        []struct {
				Value struct {
					DataPointService struct {
						SetDataPointMethod struct {
							CallMethod struct {
								Code    graphql.Int    `graphql:"code"`
								Message graphql.String `graphql:"message"`
							} `graphql:"callMethod(value: $callValue)"`
						} `graphql:"SetDataPointMethod"`
					} `graphql:"DataPointService"`
				} `graphql:"value"`
			} `graphql:"inputs(key: $inputKey)"`
		} `graphql:"Channels(find: $channelFind)"`
	} `graphql:"IDeviceFH(find: $deviceFind)"`
}

func SetDataPointValue(httpClient *http.Client, serialNumber string, channel int, datapoint string, value any) error {
	var query setQuery
	variables := map[string]interface{}{
		"deviceFind":  graphql.String(fmt.Sprintf("{'serialNumber': '%s'}", serialNumber)),
		"channelFind": graphql.String(fmt.Sprintf("{'channelNumber': %d}", channel)),
		"inputKey":    graphql.String(datapoint),
		"callValue":   graphql.String(fmt.Sprintf("%v", value)),
	}

	client := getClient(httpClient)
	if err := client.Query(context.Background(), &query, variables); err != nil {
		return fmt.Errorf("querying: %v", err)
	}

	// Check for errors
	for _, device := range query.IDeviceFH {
		for _, channel := range device.Channels {
			for _, input := range channel.Inputs {
				cm := input.Value.DataPointService.SetDataPointMethod.CallMethod
				if cm.Code >= 300 {
					return fmt.Errorf("setting data point on device %v channel %v input %v: %v (%v)", serialNumber, channel, datapoint, cm.Code, cm.Message)
				}
			}
		}
	}
	return nil
}

//

func getClient(httpClient *http.Client) *graphql.Client {
	return graphql.NewClient("https://apim.eu.mybuildings.abb.com/adtg-api/v1/graphql", httpClient)
}
