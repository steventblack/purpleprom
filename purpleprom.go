package main

import (
	"log"
	"time"
)

func main() {
	conf := configLoad("purpleprom.conf")
	//	log.Printf("conf: %v", conf)

	if conf.Metrics.Enabled {
		metricsDisplay(conf.Metrics.Path, conf.Metrics.Port)
	}

	for {
		for _, s := range conf.Sensors {
			r, err := sensorRead(s)
			if err != nil {
				log.Fatal(err.Error())
			}

			//			log.Printf("sensor %d: %+v", s, r)
			metricsUpdate(r.Results)
		}

		time.Sleep(conf.PollInterval.Duration())
	}
}
