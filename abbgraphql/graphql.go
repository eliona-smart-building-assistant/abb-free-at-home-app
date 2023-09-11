package abbgraphql

import (
	"context"
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
			} `graphql:"Channels"`
		} `graphql:"Assets"`
	} `graphql:"ISystemFH"`
}

func GetSystems(httpClient *http.Client) (SystemsQuery, error) {
	client := graphql.NewClient("https://apim.eu.mybuildings.abb.com/adtg-api/v1/graphql", httpClient)

	var query SystemsQuery
	variables := map[string]interface{}{
		//"channelNameEn": graphql.String("Ⓒ"),
	}
	if err := client.Query(context.Background(), &query, variables); err != nil {
		return SystemsQuery{}, err
	}
	return query, nil
}
