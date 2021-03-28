package config_test

import (
	"testing"

	"github.com/Stoakes/go-pkg/config"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type ConfigurationTest struct {
	Mode             bool   `hcl:"mode" default:"false" comment:"Use remote or local as backend"`
	ConnectionString string `hcl:"connection_string" default:"postgresql://user:password@localhost:5432/postgres" comment:"Database connection string"`
}

// TestLoaderNoOpts test loading a configuration without any additional options
func TestLoaderNoOpts(t *testing.T) {
	conf := &ConfigurationTest{}
	err := config.Load(conf, "AAAA", "")
	if err != nil {
		t.Fatal("Cannot load test configuration: " + err.Error())
	}
}

// TestLoaderWithOpts tests loading a configuration with additional viper.DecodeHook
func TestLoaderWithOpts(t *testing.T) {
	conf := &ConfigurationTest{}
	configOptions := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	))
	err := config.Load(conf, "BBBB", "", configOptions)
	if err != nil {
		t.Fatal("Cannot load test configuration: " + err.Error())
	}
}
