package abbgraphql

import (
	"abb-free-at-home/appdb"
	"context"
	"fmt"
	"net/http"

	graphql "github.com/hasura/go-graphql-client"
	"github.com/hasura/go-graphql-client/pkg/jsonutil"
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

type DataPoint struct {
	Value         graphql.String `graphql:"value"`
	SerialNumber  graphql.String `graphql:"serialNumber"`
	ChannelNumber graphql.String `graphql:"channelNumber"`
	DatapointId   graphql.String `graphql:"datapointId"`
}

type DataPointsSubscription struct {
	DataPointsSubscription DataPoint `graphql:"DataPointsSubscription(datapointList: $datapointList)"`
}

func SubscribeDataPointValue(authToken string, datapoints []appdb.Datapoint) error {
	client := graphql.NewSubscriptionClient("wss://apps.eu.mybuildings.abb.com/adtg-ws/graphql").
		WithConnectionParams(map[string]interface{}{
			"authorization": "Bearer " + authToken,
		}).
		WithProtocol(graphql.GraphQLWS).
		OnError(func(sc *graphql.SubscriptionClient, err error) error {
			// Cancels the subscription if returns non-nil error.
			return fmt.Errorf("subscription client error: %v", err)
		})
	defer client.Close()

	type DataPointSubscriptionArgs map[string]string
	datapointsList := []DataPointSubscriptionArgs{}
	for _, dp := range datapoints {
		datapointsList = append(datapointsList, map[string]string{
			"dtId":          dp.SystemID,
			"serialNumber":  dp.DeviceID,
			"channelNumber": dp.ChannelID,
			"datapointId":   dp.Datapoint,
		})
	}

	var sub DataPointsSubscription

	variables := map[string]interface{}{
		"datapointList": datapointsList,
	}

	// Useful for debugging
	// jsonVars, err := json.MarshalIndent(variables, "", "    ")
	// if err != nil {
	// 	fmt.Println("Error encoding to JSON:", err)
	// }
	// s1, s2, err := graphql.ConstructSubscription(sub, variables)
	// fmt.Printf("%v\n%v\n%v\n%v\n", s1, s2, err, string(jsonVars))
	// return nil

	if _, err := client.Subscribe(&sub, variables, func(message []byte, err error) error {
		if err != nil {
			return fmt.Errorf("subscribe: %v", err)
		}
		data := DataPointsSubscription{}
		if err := jsonutil.UnmarshalGraphQL(message, &data); err != nil {
			return fmt.Errorf("unmarshalling subscription response: %v", err)
		}
		fmt.Println(data)
		return nil
	}, nil); err != nil {
		return fmt.Errorf("establishing subscription: %v", err)
	}

	if err := client.Run(); err != nil {
		return fmt.Errorf("running client: %v", err)
	}
	fmt.Println("client run")
	return nil
}

//

func getClient(httpClient *http.Client) *graphql.Client {
	return graphql.NewClient("https://apim.eu.mybuildings.abb.com/adtg-api/v1/graphql", httpClient)
}
