package main

import (
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/stretchr/testify/assert"
)

func TestReadConfig(t *testing.T) {
	assert := assert.New(t)

	t.Run("pass", func(t *testing.T) {
		out, err := readConfigFile("./testdata/config.pass.yml")
		if assert.Nil(err) {
			assert.Equal("oct_token", out.Octopus.Token)
			assert.Equal("http://localhost:8086", out.Influx.URL)
			assert.Equal("token", out.Influx.Token)

			assert.Equal("12345", out.Elec.MPAN)
			assert.Equal("12345", out.Elec.Serial)

			assert.Equal("12345", out.Gas.MPAN)
			assert.Equal("12345", out.Gas.Serial)
		}
	})

	t.Run("fail_open", func(t *testing.T) {
		_, err := readConfigFile(filepath.Join(os.TempDir(), strconv.Itoa(rand.Intn(1e6)), ".yml"))
		if assert.NotNil(err) {
			assert.Contains(err.Error(), "open")
		}
	})

	t.Run("fail_parse", func(t *testing.T) {
		_, err := readConfigFile("./testdata/config.fail.yml")

		if assert.NotNil(err) {
			assert.Contains(err.Error(), "parse")
		}
	})

	t.Run("fail_influx", func(t *testing.T) {
		_, err := readConfigFile("./testdata/config.fail_influx.yml")

		if assert.NotNil(err) {
			assert.Contains(err.Error(), "InfluxDB")
		}
	})

	t.Run("fail_token", func(t *testing.T) {
		_, err := readConfigFile("./testdata/config.fail_token.yml")

		if assert.NotNil(err) {
			assert.Contains(err.Error(), "API token")
		}
	})
}

func TestGetLastTime(t *testing.T) {
	assert := assert.New(t)

	t.Run("fail_fuel", func(t *testing.T) {
		_, err := getLastTime(nil, nil, "hydrogen")

		if assert.NotNil(err) {
			assert.Contains(err.Error(), "fuel type")
		}
	})

	t.Run("fail_influx", func(t *testing.T) {
		cfg := Config{}
		cfg.Influx.Database = "test"
		iClient := influxdb2.NewClient("http://", "test")

		_, err := getLastTime(&cfg, iClient, fuelELEC)

		if assert.NotNil(err) {
			assert.Contains(err.Error(), "database")
		}
	})
}
