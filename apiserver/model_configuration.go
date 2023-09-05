/*
 * ABB Free@Home App API
 *
 * API to access and configure the ABB Free@Home App
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package apiserver

import (
	"time"
)

// Configuration - Each configuration defines access to provider's API.
type Configuration struct {

	// Internal identifier for the configured API (created automatically).
	Id *int64 `json:"id,omitempty"`

	// Set if this API is in myBuildings cloud or a local installation.
	IsCloud bool `json:"isCloud,omitempty"`

	// ABB Cloud API OAuth client ID.
	ClientID *string `json:"clientID,omitempty"`

	// ABB Cloud API OAuth client secret.
	ClientSecret *string `json:"clientSecret,omitempty"`

	// ABB Cloud API OAuth current access token. Should not be needed to change.
	AccessToken *string `json:"accessToken,omitempty"`

	// ABB Cloud API OAuth client refresh token. Should not be needed to change.
	RefreshToken *string `json:"refreshToken,omitempty"`

	// ABB Cloud API OAuth client expiry time. Should not be needed to change.
	Expiry *time.Time `json:"expiry,omitempty"`

	// URL of the local ABB API.
	ApiUrl *string `json:"apiUrl,omitempty"`

	// ABB local API username.
	ApiUsername *string `json:"apiUsername,omitempty"`

	// ABB local API password.
	ApiPassword *string `json:"apiPassword,omitempty"`

	// Flag to enable or disable fetching from this API
	Enable *bool `json:"enable,omitempty"`

	// Interval in seconds for collecting data from API
	RefreshInterval int32 `json:"refreshInterval,omitempty"`

	// Timeout in seconds
	RequestTimeout *int32 `json:"requestTimeout,omitempty"`

	// Array of rules combined by logical OR
	AssetFilter [][]FilterRule `json:"assetFilter,omitempty"`

	// Set to `true` by the app when running and to `false` when app is stopped
	Active *bool `json:"active,omitempty"`

	// List of Eliona project ids for which this device should collect data. For each project id all smart devices are automatically created as an asset in Eliona. The mapping between Eliona is stored as an asset mapping in the KentixONE app.
	ProjectIDs *[]string `json:"projectIDs,omitempty"`
}

// AssertConfigurationRequired checks if the required fields are not zero-ed
func AssertConfigurationRequired(obj Configuration) error {
	if err := AssertRecurseFilterRuleRequired(obj.AssetFilter); err != nil {
		return err
	}
	return nil
}

// AssertRecurseConfigurationRequired recursively checks if required fields are not zero-ed in a nested slice.
// Accepts only nested slice of Configuration (e.g. [][]Configuration), otherwise ErrTypeAssertionError is thrown.
func AssertRecurseConfigurationRequired(objSlice interface{}) error {
	return AssertRecurseInterfaceRequired(objSlice, func(obj interface{}) error {
		aConfiguration, ok := obj.(Configuration)
		if !ok {
			return ErrTypeAssertionError
		}
		return AssertConfigurationRequired(aConfiguration)
	})
}
