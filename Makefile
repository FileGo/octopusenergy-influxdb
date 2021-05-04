build:
	go build -o octopusenergy-influxdb .

test:
	go test -race -v ./...

clean:
	rm -f octopusenergy-influxdb
