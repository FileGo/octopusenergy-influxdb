package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/FileGo/octopusenergyapi"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"gopkg.in/yaml.v2"
)

// Config stores program configurations
type Config struct {
	Octopus struct {
		Token string `yaml:"token"`
	} `yaml:"octopusenergy"`
	Influx struct {
		URL      string `yaml:"url"`
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
	} `yaml:"gas"`
}

const configFile = "config.yml"

func main() {
	// Check if config file exists
	_, err := os.Stat(configFile)
	if errors.Is(err, os.ErrNotExist) {
		// File doesn't exist
		log.Fatal("config.yml not found")
	}

	// Open file
	f, err := os.Open(configFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Verify config
	if cfg.Octopus.Token == "" {
		log.Fatal("No API key provided")
	}
	if cfg.Influx.URL == "" {
		log.Fatal("No URL to InfluxDB provided")
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
	writeAPI := iClient.WriteAPI("", cfg.Influx.Database)
	queryAPI := iClient.QueryAPI("")

	// Set up Octopus client
	client, err := octopusenergyapi.NewClient(cfg.Octopus.Token, http.DefaultClient)
	if err != nil {
		log.Fatal(err)
	}

	if getElec {
		// Get latest electricity entry
		res, err := queryAPI.Query(context.Background(), `from(bucket:"`+cfg.Influx.Database+`")
|> range(start: -8760h)
|> filter(fn: (r) =>
	r._measurement == "electricity" and r._value != 0
)
|> last()`)

		if err != nil {
			log.Fatalf("Unable to read from database: %v", err)
		}

		// Set last time
		var lastElecTime time.Time

		if !res.Next() {
			log.Println("Unable to read last entry from database, collecting data for last 5 years")
			lastElecTime = time.Now().Add(-5 * 365 * 24 * time.Hour) // 5 years ago
		} else {
			lastElecTime = res.Record().Time()
		}

		// Get data
		rows, err := client.GetElecMeterConsumption(cfg.Elec.MPAN, cfg.Elec.Serial, octopusenergyapi.ConsumptionOption{
			From:     lastElecTime,
			To:       time.Now(),
			OrderBy:  "period",
			PageSize: 1e6,
		})
		if err != nil {
			log.Fatal(err)
		}

		for _, row := range rows {
			p := influxdb2.NewPoint("electricity", map[string]string{"unit": "kWh"}, map[string]interface{}{"consumption": row.Value}, row.IntervalStart)
			writeAPI.WritePoint(p)
		}
	}

	if getGas {
		// Get latest gas entry
		res, err := queryAPI.Query(context.Background(), `from(bucket:"`+cfg.Influx.Database+`")
|> range(start: -8760h)
|> filter(fn: (r) =>
	r._measurement == "gas" and r._value != 0
)
|> last()`)

		if err != nil {
			log.Fatalf("Unable to read from database: %v", err)
		}

		var lastGasTime time.Time

		if !res.Next() {
			log.Println("Unable to read last entry from database, collecting data for last 5 years")
			lastGasTime = time.Now().Add(-5 * 365 * 24 * time.Hour) // 5 years ago
		} else {
			lastGasTime = res.Record().Time()
		}

		// Get data
		rows, err := client.GetGasMeterConsumption(cfg.Gas.MPAN, cfg.Gas.Serial, octopusenergyapi.ConsumptionOption{
			From:     lastGasTime,
			To:       time.Now(),
			OrderBy:  "period",
			PageSize: 1e6,
		})
		if err != nil {
			log.Fatal(err)
		}

		for _, row := range rows {
			p := influxdb2.NewPoint("gas", map[string]string{"unit": "m3"}, map[string]interface{}{"consumption": row.Value}, row.IntervalStart)
			writeAPI.WritePoint(p)
		}
	}

	writeAPI.Flush()
	iClient.Close()
	fmt.Println("Data succcessfully retrieved.")
}
