package repo

import (
	"github.com/phoreproject/openbazaar-go/schema"
	"reflect"
	"testing"

	"io/ioutil"
	"time"
)

const testConfigPath = "testdata/config"

func TestGetApiConfig(t *testing.T) {
	configFile, err := ioutil.ReadFile(testConfigPath)
	if err != nil {
		t.Error(err)
	}

	config, err := schema.GetAPIConfig(configFile)
	if config.Username != "TestUsername" {
		t.Error("Expected TestUsername, got ", config.Username)
	}
	if config.Password != "TestPassword" {
		t.Error("Expected TestPassword, got ", config.Password)
	}
	if !config.Authenticated {
		t.Error("Expected Authenticated = true")
	}
	if len(config.AllowedIPs) != 1 || config.AllowedIPs[0] != "127.0.0.1" {
		t.Error("Expected AllowedIPs = [127.0.0.1]")
	}
	if config.CORS == nil {
		t.Error("Cors is not set")
	}
	if reflect.ValueOf(config.HTTPHeaders).Kind() != reflect.Map {
		t.Error("Headers is not a map")
	}
	if config.Enabled != true {
		t.Error("Enabled is not true")
	}
	if !config.SSL {
		t.Error("Expected SSL = true")
	}
	if config.SSLCert == "" {
		t.Error("Expected test SSL cert, got ", config.SSLCert)
	}
	if config.SSLKey == "" {
		t.Error("Expected test SSL key, got ", config.SSLKey)
	}
	if err != nil {
		t.Error("GetAPIAuthentication threw an unexpected error")
	}

	_, err = schema.GetAPIConfig([]byte{})
	if err == nil {
		t.Error("GetAPIAuthentication didn`t throw an error")
	}
}

func TestGetWalletConfig(t *testing.T) {
	configFile, err := ioutil.ReadFile(testConfigPath)
	if err != nil {
		t.Error(err)
	}
	config, err := schema.GetWalletConfig(configFile)
	if err != nil {
		t.Error(err)
	}
	if config.RPCLocation != "rpc.phore.io" {
		t.Error("RPCLocation does not equal expected value")
	}
	if config.Type != "phored" {
		t.Error("Type does not equal expected value")
	}
	_, err = schema.GetWalletConfig([]byte{})
	if err == nil {
		t.Error("GetFeeAPI didn't throw an error")
	}
}

func TestGetDropboxApiToken(t *testing.T) {
	configFile, err := ioutil.ReadFile(testConfigPath)
	if err != nil {
		t.Error(err)
	}
	dropboxApiToken, err := schema.GetDropboxApiToken(configFile)
	if dropboxApiToken != "dropbox123" {
		t.Error("dropboxApiToken does not equal expected value")
	}
	if err != nil {
		t.Error("GetDropboxApiToken threw an unexpected error")
	}

	dropboxApiToken, err = schema.GetDropboxApiToken([]byte{})
	if dropboxApiToken != "" {
		t.Error("Expected empty string, got ", dropboxApiToken)
	}
	if err == nil {
		t.Error("GetDropboxApiToken didn't throw an error")
	}
}

func TestRepublishInterval(t *testing.T) {
	configFile, err := ioutil.ReadFile(testConfigPath)
	if err != nil {
		t.Error(err)
	}
	interval, err := schema.GetRepublishInterval(configFile)
	if interval != time.Hour*24 {
		t.Error("RepublishInterval does not equal expected value")
	}
	if err != nil {
		t.Error("RepublishInterval threw an unexpected error")
	}

	interval, err = schema.GetRepublishInterval([]byte{})
	if interval != time.Second*0 {
		t.Error("Expected zero duration, got ", interval)
	}
	if err == nil {
		t.Error("GetRepublishInterval didn't throw an error")
	}
}

func TestGetResolverConfig(t *testing.T) {
	configFile, err := ioutil.ReadFile(testConfigPath)
	if err != nil {
		t.Error(err)
	}
	resolvers, err := schema.GetResolverConfig(configFile)
	if err != nil {
		t.Error("GetResolverUrl threw an unexpected error")
	}
	if resolvers.Id != "https://resolver.onename.com/" {
		t.Error("resolverUrl does not equal expected value")
	}
}
