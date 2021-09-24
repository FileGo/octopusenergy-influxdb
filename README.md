# octopusenergy-influxdb

[![Go Report Card](https://goreportcard.com/badge/github.com/FileGo/octopusenergy-influxdb)](https://goreportcard.com/report/github.com/FileGo/octopusenergy-influxdb) [![Go Coverage](https://codecov.io/github/FileGo/octopusenergy-influxdb/coverage.svg?branch=main)](https://codecov.io/github/FileGo/octopusenergy-influxdb/?branch=main) ![build](https://github.com/FileGo/octopusenergy-influxdb/workflows/build/badge.svg) ![tests](https://github.com/FileGo/octopusenergy-influxdb/workflows/tests/badge.svg)

This little piece of program pulls data via Octopus Energy API into InfluxDB for onwards use.

## Configuration 
Configuration file `config.yml` is required:
```yaml
octopusenergy:
  token: "sk_live_......"

influxdb:
  url: "http://localhost:8086"
  # org: "Utilities"
  database: "octopusenergy"
  token: ""
  
electricity:
  mpan: "..."
  serial: "..."
  
gas:
  mpan: "...."
  serial: "..."
```
*Note: InfluxDB authorization token is optional.*

*Note: Organization is required for InfluxDB v2.*

If you don't wish to get electricity or gas data, just remove that part of configuration file.

Tested with InfluxDB v1.8.3 - compatibility with InfluxDB 2.0 not guaranteed.

## Usage
```shell  
$ git clone https://github.com/FileGo/octopusenergy-influxdb.git
$ cd octopusenergy-influxdb
$ make
$ ./octopusenergy-influxdb
Data succcessfully retrieved.
```
