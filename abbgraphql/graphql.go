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
			DtId         string `graphql:"dtId"`
			Label        string `graphql:"label"`
			Level        string `graphql:"level"`
			Sublocations []struct {
				DtId  string `graphql:"dtId"`
				Label string `graphql:"label"`
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
		DtId   string `graphql:"dtId"`
		Assets []struct {
			IsLocated struct {
				DtId string `graphql:"dtId"`
			} `graphql:"IsLocated"`
			SerialNumber string `graphql:"serialNumber"`
			Label        string `graphql:"label"` // Custom name set on sysAP
			Name         struct {
				En string `graphql:"en"`
			} `graphql:"Name"`
			DeviceFHRF struct {
				BatteryStatus     string `graphql:"batteryStatus"`
				AttributesService struct {
					Connectivity string `graphql:"get(key:\"connectivity\")"`
				} `graphql:"AttributesService"`
			} `graphql:"... on IDeviceFHRF"`
			Channels []struct {
				ChannelNumber int      `graphql:"channelNumber"`
				FunctionId    string   `graphql:"functionId"`
				Label         string   `graphql:"label"` // Custom name set on sysAP
				Name          struct { // Static name, based on channel type
					En string `graphql:"en"`
				} `graphql:"Name"`
				Outputs []struct {
					Key   string `graphql:"key"`
					Value struct {
						PairingId        string `graphql:"pairingId"`
						Dpt              string `graphql:"dpt"`
						DataPointService struct {
							RequestDataPointValue struct {
								Value string `graphql:"value"`
								Time  string `graphql:"time"`
							} `graphql:"RequestDataPointValue"`
						} `graphql:"DataPointService"`
					} `graphql:"value"`
				} `graphql:"outputs"`
				Inputs []struct {
					Key   string `graphql:"key"`
					Value struct {
						PairingId string `graphql:"pairingId"`
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
		"channelFind": fmt.Sprintf("{'functionId': {'$in': %s}}", formatSlice(model.GetFunctionIDsList())),
	}
	if err := client.Query(context.Background(), &query, variables); err != nil {
		return SystemsQuery{}, err
	}
	if orgUUID != "" {
		for _, system := range query.Systems {
			if err := createUserIfNotExists(client, orgUUID, system.DtId); err != nil {
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
			ChannelNumber int `graphql:"channelNumber"`
			Inputs        []struct {
				Value struct {
					DataPointService struct {
						SetDataPointMethod struct {
							CallMethod struct {
								Code    int    `graphql:"code"`
								Message string `graphql:"details"`
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
			ChannelNumber int `graphql:"channelNumber"`
			Inputs        []struct {
				Value struct {
					DataPointService struct {
						SetDataPointMethod struct {
							CallMethod struct {
								Code    int    `graphql:"code"`
								Message string `graphql:"details"`
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
		"deviceFind":  fmt.Sprintf("{'serialNumber': '%s'}", serialNumber),
		"channelFind": fmt.Sprintf("{'channelNumber': %d}", channel),
		"inputKey":    datapoint,
		"callValue":   val,
	}

	// TODO: This is ugly, but doesn't work with type casting. We should find a nicer solution.
	if isProService {
		query := setQueryProService{}
		variables["orgUser"] = proServiceUser

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
	Value         string `graphql:"value"`
	SerialNumber  string `graphql:"serialNumber"`
	ChannelNumber string `graphql:"channelNumber"`
	DatapointId   string `graphql:"datapointId"`
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
// 		DtId string `graphql:"dtId"`
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
// 		if err := createUserIfNotExists(client, system.DtId); err != nil {
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
					Code    int
					Details string
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
		"dtId":        []string{dtId},
		"displayName": "Eliona ProService",
		"user":        username,
		"scopes":      []string{"RemoteControl"},
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
			UserName string
		} `graphql:"Users"`
	} `graphql:"ISystemFH(dtId: $dtId)"`
}

func userExists(client *graphql.Client, dtId string, userName string) (bool, error) {
	var query usersQuery
	variables := map[string]interface{}{
		"dtId": []string{dtId},
	}

	if err := client.Query(context.Background(), &query, variables); err != nil {
		return false, fmt.Errorf("executing query: %v", err)
	}

	for _, system := range query.ISystemFH {
		for _, user := range system.Users {
			if user.UserName == userName {
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
