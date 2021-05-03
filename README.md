# octopusenergy-influxdb
This little piece of program pulls data via Octopus Energy API into InfluxDB for onwards use.

Configuration file `config.yml` is required:
```
octopusenergy:
  token: "sk_live_......"

influxdb:
  url: "http://localhost:8086"
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

If you don't wish to get electricity or gas data, just remove that part of configuration file.

Tested with InfluxDB v1.8.3 - compatibility with InfluxDB 2.0 not guaranteed.

```
$ ./octopusenergy-influxdb
 
Data succcessfully retrieved.
```