package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"math"
	"net/http"
	"strconv"
)

var (
	pamTempVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_temp",
		Help: "PurpleAir temperature (F) reading."},
		[]string{"sensor"})

	pamHumidityVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_humidity",
		Help: "PurpleAir humidity reading."},
		[]string{"sensor"})

	pamPressureVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_pressure",
		Help: "PurpleAir pressure reading."},
		[]string{"sensor"})

	pamPm25Vec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_pm_2_5",
		Help: "PurpleAir PM 2.5 ug/m3 reading."},
		[]string{"sensor"})

	pamPm100Vec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_pm_10_0",
		Help: "PurpleAir PM 10.0 ug/m3 reading."},
		[]string{"sensor"})

	pamPm25AQIVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_AQI_pm_2_5",
		Help: "PurpleAir AQI calculation based on PM 2.5 ug/m3 reading."},
		[]string{"sensor"})

	pamPm100AQIVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_AQI_pm_10_0",
		Help: "PurpleAir AQI calculation based on PM 10.0 ug/m3 reading."},
		[]string{"sensor"})

	pamAQIVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_AQI",
		Help: "PurpleAir AQI calculation based on all available inputs"},
		[]string{"sensor"})
)

// metricsUpdate takes the slice of paSensorResults and extracts the pertinent readings
// into the prometheus collectors. PurpleAir sensors may have a primary and secondary
// sensor bundled together and so there may be multiple results per reading. However,
// the secondary sensor does not have the full set of data represented in the primary
// (e.g. Temp, Humidity, Pressure) which metricsUpdate will correctly omit.
func metricsUpdate(results []paSensorResult) {
	for _, r := range results {
		// preconvert this to a string for convenience
		sensorId := strconv.Itoa(r.Id)

		// only the primary sensor has valid entries for temp, humidity, pressure
		// in order to avoid omitting data on a cold day, check to see if
		// they are all set to 0's to determine validity.
		if r.Temp != 0 && r.Humidity != 0 && r.Pressure != 0 {
			pamTempVec.WithLabelValues(sensorId).Set(float64(r.Temp))
			pamHumidityVec.WithLabelValues(sensorId).Set(float64(r.Humidity))
			pamPressureVec.WithLabelValues(sensorId).Set(r.Pressure)
		}

		// publish the raw readings
		pamPm25Vec.WithLabelValues(sensorId).Set(r.Pm25_cf1)
		pamPm100Vec.WithLabelValues(sensorId).Set(r.Pm100_cf1)

		// calc the individual AQIs for the various measurements
		aqi_pm25 := sensorAQI(r.Pm25_cf1)
		aqi_pm100 := sensorAQI(r.Pm100_cf1)

		// publish the individual AQI calculations
		pamPm25AQIVec.WithLabelValues(sensorId).Set(aqi_pm25)
		pamPm100AQIVec.WithLabelValues(sensorId).Set(aqi_pm100)

		// publish the calculated AQI (max of all AQI calculations)
		aqi := math.Max(aqi_pm25, aqi_pm100)
		pamAQIVec.WithLabelValues(sensorId).Set(aqi)
	}
}

// metricsDisplay exports the collected metrics in the standard web interface format
// using the path and port information provided. This can be scraped by the usual
// Prometheus collection mechansims.
func metricsDisplay(path string, port int) {
	if path == "" || port == 0 {
		log.Println("Metrics export disabled")
		return
	}

	http.Handle(path, promhttp.Handler())
	portStr := ":" + strconv.Itoa(port)

	go func() {
		http.ListenAndServe(portStr, nil)
	}()
}
