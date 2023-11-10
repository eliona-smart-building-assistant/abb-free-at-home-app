package abbgraphql

import (
	"abb-free-at-home/appdb"
	"abb-free-at-home/model"
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/eliona-smart-building-assistant/go-utils/log"
	graphql "github.com/hasura/go-graphql-client"
	"github.com/hasura/go-graphql-client/pkg/jsonutil"
)

const proServiceUser = "eliona"

type LocationsQuery struct {
	ISystemFH []struct {
		Locations []struct {
			DtId         graphql.String `graphql:"dtId"`
			Label        graphql.String `graphql:"label"`
			Level        graphql.String `graphql:"level"`
			Sublocations []struct {
				DtId  graphql.String `graphql:"dtId"`
				Label graphql.String `graphql:"label"`
			} `graphql:"Sublocations"`
		} `graphql:"Locations"`
	} `graphql:"ISystemFH"`
}

func GetLocations(httpClient *http.Client) (LocationsQuery, error) {
	client := getClient(httpClient)
	var query LocationsQuery
	variables := map[string]interface{}{}
	if err := client.Query(context.Background(), &query, variables); err != nil {
		return LocationsQuery{}, err
	}
	return query, nil
}

type SystemsQuery struct {
	Systems []struct {
		DtId   graphql.String `graphql:"dtId"`
		Assets []struct {
			IsLocated struct {
				DtId graphql.String `graphql:"dtId"`
			} `graphql:"IsLocated"`
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
					} `graphql:"value"`
				} `graphql:"inputs"`
			} `graphql:"Channels(find:$channelFind, selective:false)"`
		} `graphql:"Assets"`
	} `graphql:"ISystemFH"`
}

func GetSystems(httpClient *http.Client, orgUUID string) (SystemsQuery, error) {
	client := getClient(httpClient)
	var query SystemsQuery
	variables := map[string]interface{}{
		// Fetch only supported devices.
		"channelFind": graphql.String(fmt.Sprintf("{'functionId': {'$in': %s}}", formatSlice(model.GetFunctionIDsList()))),
	}
	if err := client.Query(context.Background(), &query, variables); err != nil {
		return SystemsQuery{}, err
	}
	if orgUUID != "" {
		for _, system := range query.Systems {
			if err := createUserIfNotExists(client, orgUUID, string(system.DtId)); err != nil {
				return SystemsQuery{}, fmt.Errorf("creating user if not exists: %v", err)
			}
		}
	}
	return query, nil
}

func formatSlice(slice []string) string {
	var formattedSlice []string
	for _, s := range slice {
		formattedSlice = append(formattedSlice, fmt.Sprintf("'%s'", s))
	}
	return "[" + strings.Join(formattedSlice, ",") + "]"
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
								Message graphql.String `graphql:"details"`
							} `graphql:"callMethod(value: $callValue)"`
						} `graphql:"SetDataPointMethod"`
					} `graphql:"DataPointService"`
				} `graphql:"value"`
			} `graphql:"inputs(key: $inputKey)"`
		} `graphql:"Channels(find: $channelFind)"`
	} `graphql:"IDeviceFH(find: $deviceFind)"`
}

type setQueryProService struct {
	IDeviceFH []struct {
		Channels []struct {
			ChannelNumber graphql.Int `graphql:"channelNumber"`
			Inputs        []struct {
				Value struct {
					DataPointService struct {
						SetDataPointMethod struct {
							CallMethod struct {
								Code    graphql.Int    `graphql:"code"`
								Message graphql.String `graphql:"details"`
							} `graphql:"callMethod(value: $callValue, setOrgUser: $orgUser)"`
						} `graphql:"SetDataPointMethod"`
					} `graphql:"DataPointService"`
				} `graphql:"value"`
			} `graphql:"inputs(key: $inputKey)"`
		} `graphql:"Channels(find: $channelFind)"`
	} `graphql:"IDeviceFH(find: $deviceFind)"`
}

func SetDataPointValue(httpClient *http.Client, isProService bool, serialNumber string, channel int, datapoint string, value float64) error {
	client := getClient(httpClient)
	val := formatFloat(value)
	variables := map[string]interface{}{
		"deviceFind":  graphql.String(fmt.Sprintf("{'serialNumber': '%s'}", serialNumber)),
		"channelFind": graphql.String(fmt.Sprintf("{'channelNumber': %d}", channel)),
		"inputKey":    graphql.String(datapoint),
		"callValue":   graphql.String(val),
	}

	// TODO: This is ugly, but doesn't work with type casting. We should find a nicer solution.
	if isProService {
		query := setQueryProService{}
		variables["orgUser"] = graphql.String(proServiceUser)

		if err := client.Query(context.Background(), &query, variables); err != nil {
			return fmt.Errorf("querying: %v", err)
		}

		// Check for errors
		for _, device := range query.IDeviceFH {
			for _, channel := range device.Channels {
				for _, input := range channel.Inputs {
					methodCall := input.Value.DataPointService.SetDataPointMethod.CallMethod
					if methodCall.Code >= 300 {
						return fmt.Errorf("setting data point value %s on device %v channel %v input %v: %v (%v)", val, serialNumber, channel, datapoint, methodCall.Code, methodCall.Message)
					}
				}
			}
		}
	} else {
		query := setQuery{}
		if err := client.Query(context.Background(), &query, variables); err != nil {
			return fmt.Errorf("querying: %v", err)
		}

		// Check for errors
		for _, device := range query.IDeviceFH {
			for _, channel := range device.Channels {
				for _, input := range channel.Inputs {
					methodCall := input.Value.DataPointService.SetDataPointMethod.CallMethod
					if methodCall.Code >= 300 {
						return fmt.Errorf("setting data point value %s on device %v channel %v input %v: %v (%v)", val, serialNumber, channel, datapoint, methodCall.Code, methodCall.Message)
					}
				}
			}
		}
	}

	return nil
}

func formatFloat(f float64) string {
	if f == math.Trunc(f) {
		return fmt.Sprintf("%.0f", f)
	}
	return fmt.Sprintf("%f", f)
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

func SubscribeDataPointValue(auth string, datapoints []appdb.Datapoint, ch chan<- DataPoint) error {
	client := graphql.NewSubscriptionClient("wss://apps.eu.mybuildings.abb.com/adtg-ws/graphql").
		WithConnectionParams(map[string]interface{}{
			"authorization": auth,
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
		ch <- data.DataPointsSubscription
		return nil
	}); err != nil {
		return fmt.Errorf("establishing subscription: %v", err)
	}

	if err := client.Run(); err != nil {
		return fmt.Errorf("running client: %v", err)
	}
	close(ch)
	return nil
}

// type systemsSimpleQuery struct {
// 	Systems []struct {
// 		DtId graphql.String `graphql:"dtId"`
// 	} `graphql:"ISystemFH"`
// }

// func EnsureAllUsersAreCreated(httpClient *http.Client) error {
// 	client := getClient(httpClient)
// 	var systems systemsSimpleQuery
// 	variables := map[string]interface{}{}
// 	if err := client.Query(context.Background(), &systems, variables); err != nil {
// 		return fmt.Errorf("fetching list of systems: %v", err)
// 	}
// 	for _, system := range systems.Systems {
// 		if err := createUserIfNotExists(client, string(system.DtId)); err != nil {
// 			return fmt.Errorf("creatíng user if not exists: %v", err)
// 		}
// 	}
// 	return nil
// }

type createUserMutation struct {
	ISystem []struct {
		DeviceManagement struct {
			RPCCreateUserWithPermissionsMethod struct {
				CallMethod struct {
					Code    graphql.Int
					Details graphql.String
				} `graphql:"callMethod(displayName: $displayName, user: $user, scopes: $scopes)"`
			} `graphql:"RPCCreateUserWithPermissionsMethod"`
		} `graphql:"DeviceManagement"`
	} `graphql:"ISystem(dtId: $dtId)"`
}

func createUserIfNotExists(client *graphql.Client, orgUUID, dtId string) error {
	username := fmt.Sprintf("%s_%s", orgUUID, proServiceUser)
	if exists, err := userExists(client, dtId, username); err != nil {
		return fmt.Errorf("checking if ProService user exists: %v", err)
	} else if exists {
		return nil
	}

	var mutation createUserMutation
	variables := map[string]interface{}{
		"dtId":        []graphql.String{graphql.String(dtId)},
		"displayName": graphql.String("Eliona ProService"),
		"user":        graphql.String(username),
		"scopes":      []graphql.String{graphql.String("RemoteControl")},
	}

	if err := client.Query(context.Background(), &mutation, variables); err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	for _, system := range mutation.ISystem {
		if system.DeviceManagement.RPCCreateUserWithPermissionsMethod.CallMethod.Code != 204 {
			methodCall := system.DeviceManagement.RPCCreateUserWithPermissionsMethod.CallMethod
			return fmt.Errorf("API returned error code %d: %s", methodCall.Code, methodCall.Details)
		}
	}

	log.Info("GraphQL", "Created a new user %s in system %s. Please log in to that system and give write permissions to this user.", proServiceUser, dtId)
	return nil
}

type usersQuery struct {
	ISystemFH []struct {
		Users []struct {
			UserName graphql.String
		} `graphql:"Users"`
	} `graphql:"ISystemFH(dtId: $dtId)"`
}

func userExists(client *graphql.Client, dtId string, userName string) (bool, error) {
	var query usersQuery
	variables := map[string]interface{}{
		"dtId": []graphql.String{graphql.String(dtId)},
	}

	if err := client.Query(context.Background(), &query, variables); err != nil {
		return false, fmt.Errorf("executing query: %v", err)
	}

	for _, system := range query.ISystemFH {
		for _, user := range system.Users {
			if string(user.UserName) == userName {
				return true, nil
			}
		}
	}
	return false, nil
}

//

func getClient(httpClient *http.Client) *graphql.Client {
	return graphql.NewClient("https://apim.eu.mybuildings.abb.com/adtg-api/v1/graphql", httpClient)
}
