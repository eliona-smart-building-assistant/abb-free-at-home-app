package main

import (
	"testing"

	"github.com/eliona-smart-building-assistant/app-integration-tests/app"
	"github.com/eliona-smart-building-assistant/app-integration-tests/assert"
	"github.com/eliona-smart-building-assistant/app-integration-tests/test"
)

func TestApp(t *testing.T) {
	app.StartApp()
	test.AppWorks(t)
	t.Run("TestAssetTypes", assetTypes)
	t.Run("TestSchema", schema)
	app.StopApp()
}

func schema(t *testing.T) {
	t.Parallel()

	assert.SchemaExists(t, "abb_free_at_home", []string{"configuration", "asset", "datapoint", "datapoint_attribute"})
}

func assetTypes(t *testing.T) {
	t.Parallel()

	assert.AssetTypeExists(t, "abb_free_at_home_channel", []string{})
	assert.AssetTypeExists(t, "abb_free_at_home_device", []string{})
	assert.AssetTypeExists(t, "abb_free_at_home_dimmer_sensor", []string{})
	assert.AssetTypeExists(t, "abb_free_at_home_floor", []string{})
	assert.AssetTypeExists(t, "abb_free_at_home_heating_actuator", []string{})
	assert.AssetTypeExists(t, "abb_free_at_home_hue_actuator", []string{})
	assert.AssetTypeExists(t, "abb_free_at_home_movement_sensor", []string{})
	assert.AssetTypeExists(t, "abb_free_at_home_radiator_thermostat", []string{})
	assert.AssetTypeExists(t, "abb_free_at_home_room", []string{})
	assert.AssetTypeExists(t, "abb_free_at_home_room_temperature_controller", []string{})
	assert.AssetTypeExists(t, "abb_free_at_home_root", []string{})
	assert.AssetTypeExists(t, "abb_free_at_home_scene", []string{})
	assert.AssetTypeExists(t, "abb_free_at_home_switch_sensor", []string{})
	assert.AssetTypeExists(t, "abb_free_at_home_system", []string{})
	assert.AssetTypeExists(t, "abb_free_at_home_window_sensor", []string{})
}
