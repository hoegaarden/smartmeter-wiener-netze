package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hoegaarden/smartmeter-wiener-netze"
	"github.com/influxdata/telegraf/metric"
	"github.com/influxdata/telegraf/plugins/serializers/influx"
)

const usage = `
Pulls 15m consumption data from Wiener Netze Smartmeter API and prints it in
the influx line protocol on stdout. It does this for all smart meters in the
customer's account. To log in, 'SMARTMETER_USERNAME' and 'SMARTMETER_PASSWORD'
need to be set in the environment to the creds for log.wien.

All metrics will have a single value 'value' and are tagged with:
- the meter's ID as 'meterID'
- the meter's device ID as 'deviceID'
- the meter's equipment ID as 'equipmentID'
- the meter's label as 'customLabel'

The timestamp is the end of the 15m interval and the 'value' is the
consumption in Wh for the 15m period up until the metrics timestamp.

Data starts from the first reading on the 'start' date to the last reading on
'end' date; all times/date are in UTC. Both 'start' and 'end' are mandatory.
`

func main() {
	var startDay, endDay, metricName string

	flag.StringVar(&startDay, "start", "", "start day, in ISO 8601)")
	flag.StringVar(&endDay, "end", "", "end day, in ISO 8601)")
	flag.StringVar(&metricName, "metricName", "smartmeter", "name of the metric")

	flag.Usage = func(orgUsage func()) func() {
		return func() {
			orgUsage()
			out := flag.CommandLine.Output()
			fmt.Fprintln(out, usage)
			fmt.Fprintln(out, "Version:", smartmeter.VersionInfo())
		}
	}(flag.Usage)

	flag.Parse()
	if startDay == "" || endDay == "" {
		flag.Usage()
		os.Exit(1)
	}

	start, err := time.Parse("2006-01-02", startDay)
	if err != nil {
		panic(err)
	}

	end, err := time.Parse("2006-01-02", endDay)
	if err != nil {
		panic(err)
	}
	end = end.Add(24 * time.Hour).Truncate(24 * time.Hour).Add(-time.Microsecond)

	if _, err := fmt.Fprintf(os.Stderr, "## Export -- start: %s, end: %s\n", start, end); err != nil {
		panic(err)
	}

	username, ok := os.LookupEnv("SMARTMETER_USERNAME")
	if !ok {
		panic("SMARTMETER_USERNAME not set")
	}

	password, ok := os.LookupEnv("SMARTMETER_PASSWORD")
	if !ok {
		panic("SMARTMETER_PASSWORD not set")
	}

	client, err := smartmeter.Login(username, password)
	if err != nil {
		panic(err)
	}

	meters, err := client.Meters()
	if err != nil {
		panic(err)
	}

	serializer := &influx.Serializer{}
	if err := serializer.Init(); err != nil {
		panic(err)
	}

	for _, meter := range meters {
		export, err := client.Export(meter.CustomerID, meter.ID, start, end)
		if err != nil {
			panic(err)
		}

		for _, point := range export.Values {
			metric := metric.New(
				metricName,
				map[string]string{ // tags
					"customLabel": meter.Label,
					"deviceID":    meter.DeviceID,
					"equipmentID": meter.EquipmentID,
					"meterID":     meter.ID,
				},
				map[string]any{ // values
					"value": point.Value * 1000, // convert from kWh to Wh
				},
				point.To.Time,
			)

			if err := serializer.Write(os.Stdout, metric); err != nil {
				panic(err)
			}
		}
	}
}
