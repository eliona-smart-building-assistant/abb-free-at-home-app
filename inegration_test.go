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
	t.Run("TestSchema", schema)
	app.StopApp()
}

func schema(t *testing.T) {
	t.Parallel()

	assert.SchemaExists(t, "abb_free_at_home", []string{ /* insert tables */ })
}
