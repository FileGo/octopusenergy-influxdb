package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/FileGo/octopusenergyapi"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"gopkg.in/yaml.v3"
)

// Config stores program configurations
type Config struct {
	Octopus struct {
		Token string `yaml:"token"`
	} `yaml:"octopusenergy"`
	Influx struct {
		URL      string `yaml:"url"`
		Org      string `yaml:"org"`
		Database string `yaml:"database"`
		Token    string `yaml:"token"`
	} `yaml:"influxdb"`
	Elec struct {
		MPAN   string `yaml:"mpan"`
		Serial string `yaml:"serial"`
	} `yaml:"electricity"`
	Gas struct {
		MPAN   string `yaml:"mpan"`
		Serial string `yaml:"serial"`
		Type   string `yaml:"type"`
	} `yaml:"gas"`
}

const configFile = "config.yml"

const (
	fuelELEC = "electricity"
	fuelGAS  = "gas"
)

// readConfigFile reads configuration from a file
//
// Returns error if file doesn't exist, file cannot be parsed or fields don't pass validation
func readConfigFile(filename string) (*Config, error) {
	// Open file
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %s: %v", configFile, err)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to parse configuration: %v", err)
	}

	// Verify config
	if cfg.Octopus.Token == "" {
		return nil, errors.New("API token not provided")
	}
	if cfg.Influx.URL == "" {
		return nil, errors.New("InfluxDB URL not provided")
	}

	return &cfg, nil
}

func getLastTime(cfg *Config, iClient influxdb2.Client, fuel string) (time.Time, error) {
	if !(fuel == fuelELEC || fuel == fuelGAS) {
		return time.Time{}, errors.New("incorrect fuel type")
	}

	queryAPI := iClient.QueryAPI(cfg.Influx.Org)

	// Get latest electricity entry
	res, err := queryAPI.Query(context.Background(), `from(bucket:"`+cfg.Influx.Database+`")
	|> range(start: -8760h)
	|> filter(fn: (r) =>
		r._measurement == "`+fuel+`" and r._value != 0
	)
	|> last()`)

	if err != nil {
		return time.Time{}, fmt.Errorf("unable to read from database: %v", err)
	}

	// Set last time
	var lastTime time.Time

	if !res.Next() {
		lastTime = time.Now().Add(-5 * 365 * 24 * time.Hour) // 5 years ago
	} else {
		lastTime = res.Record().Time()
	}

	return lastTime, nil
}

func main() {
	cfg, err := readConfigFile(configFile)
	if err != nil {
		log.Fatalf("unable to read configuration file: %v", err)
	}

	var getElec, getGas bool

	if cfg.Elec.MPAN != "" && cfg.Elec.Serial != "" {
		getElec = true
	}
	if cfg.Gas.MPAN != "" && cfg.Gas.Serial != "" {
		getGas = true
	}

	if !getElec && !getGas {
		log.Fatal("Both electricity and gas meter data not provided, nothing to do here.")
	}

	// InfluxDB
	iClient := influxdb2.NewClient(cfg.Influx.URL, cfg.Influx.Token)
	writeAPI := iClient.WriteAPI(cfg.Influx.Org, cfg.Influx.Database)

	// Set up Octopus client
	client, err := octopusenergyapi.NewClient(cfg.Octopus.Token, http.DefaultClient)
	if err != nil {
		log.Fatal(err)
	}

	if getElec {
		// Get latest electricity entry
		lastElecTime, err := getLastTime(cfg, iClient, "electricity")
		if err != nil {
			log.Println("unable to get last record for electricity in InfluxDB, retrieving data for last 5 years")
			lastElecTime = time.Now().Add(-5 * 365 * 24 * time.Hour)
		}

		// Get data
		rows, err := client.GetElecMeterConsumption(cfg.Elec.MPAN, cfg.Elec.Serial, octopusenergyapi.ConsumptionOption{
			From:     lastElecTime,
			To:       time.Now(),
			OrderBy:  "period",
			PageSize: 1e6,
		})
		if err != nil {
			log.Fatalf("error reading electricity meter consumption: %v", err)
		}

		for _, row := range rows {
			p := influxdb2.NewPoint("electricity", map[string]string{"unit": "kWh"}, map[string]interface{}{"consumption": row.Value}, row.IntervalStart)
			writeAPI.WritePoint(p)
		}
	}

	if getGas {
		// Get latest gas entry
		lastGasTime, err := getLastTime(cfg, iClient, "gas")
		if err != nil {
			log.Println("unable to get last record for gas in InfluxDB, retrieving data for last 5 years")
			lastGasTime = time.Now().Add(-5 * 365 * 24 * time.Hour)
		}

		// Get data
		rows, err := client.GetGasMeterConsumption(cfg.Gas.MPAN, cfg.Gas.Serial, octopusenergyapi.ConsumptionOption{
			From:     lastGasTime,
			To:       time.Now(),
			OrderBy:  "period",
			PageSize: 1e6,
		})
		if err != nil {
			log.Fatalf("error reading gas meter consumption: %v", err)
		}

		// SMETS1 meters report kWh for gas, SMETS2 reports m3
		// See https://developer.octopus.energy/docs/api/#consumption
		unit := "m3"
		if strings.ToLower(cfg.Gas.Type) == "smets1" {
			unit = "kWh"
		}

		for _, row := range rows {
			p := influxdb2.NewPoint("gas", map[string]string{"unit": unit}, map[string]interface{}{"consumption": row.Value}, row.IntervalStart)
			writeAPI.WritePoint(p)
		}
	}

	writeAPI.Flush()
	iClient.Close()
	fmt.Println("Data succcessfully retrieved.")
}
