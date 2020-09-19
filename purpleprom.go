package main

import (
	"log"
	"time"
)

func main() {
	conf := configLoad("purpleprom.conf")
	if conf.Metrics.Enabled {
		metricsDisplay(conf.Metrics.Path, conf.Metrics.Port)
	}

	for {
		r, err := sensorRead(conf.Sensors)
		if err != nil {
			log.Print(err.Error())
		} else {
			metricsRecord(r.Results)
		}

		time.Sleep(conf.PollInterval.Duration())
	}
}
