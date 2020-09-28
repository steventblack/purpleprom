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
		[]string{"sensor", "parent"})

	pamHumidityVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_humidity",
		Help: "PurpleAir humidity reading."},
		[]string{"sensor", "parent"})

	pamPressureVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_pressure",
		Help: "PurpleAir pressure reading."},
		[]string{"sensor", "parent"})

	pamPm25Vec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_pm_2_5",
		Help: "PurpleAir PM 2.5 ug/m3 reading."},
		[]string{"sensor", "parent"})

	pamPm100Vec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_pm_10_0",
		Help: "PurpleAir PM 10.0 ug/m3 reading."},
		[]string{"sensor", "parent"})

	pamPm25AQIVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_AQI_pm_2_5",
		Help: "PurpleAir AQI calculation based on PM 2.5 ug/m3 reading."},
		[]string{"sensor", "parent"})

	pamPm100AQIVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_AQI_pm_10_0",
		Help: "PurpleAir AQI calculation based on PM 10.0 ug/m3 reading."},
		[]string{"sensor", "parent"})

	pamAQIVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_AQI",
		Help: "PurpleAir AQI calculation based on all available inputs"},
		[]string{"sensor", "parent"})

	pamLabelVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pa_label",
		Help: "PurpleAir sensor to label map."},
		[]string{"sensor", "parent", "label"})
)

// metricsRecord takes the slice of paSensorResults and extracts the pertinent readings
// into the prometheus collectors. PurpleAir sensors may have a primary and secondary
// sensor bundled together and so there may be multiple results per reading. However,
// the secondary sensor does not have the full set of data represented in the primary
// (e.g. Temp, Humidity, Pressure) which metricsUpdate will correctly omit.
func metricsRecord(results []paSensorResult) {
	for _, r := range results {
		// preconvert these to strings for convenience
		sensorId := strconv.Itoa(r.Id)
		parentId := strconv.Itoa(r.ParentId)

		// Provides means to map a sensorId to a Label value
		pamLabelVec.WithLabelValues(sensorId, parentId, r.Label).Set(1.0)

		// only the parent sensor has valid entries for temp, humidity, pressure
		// check to see if this is a parent sensor before recording metrics
		if r.Id == r.ParentId {
			pamTempVec.WithLabelValues(sensorId, parentId).Set(float64(r.Temp))
			pamHumidityVec.WithLabelValues(sensorId, parentId).Set(float64(r.Humidity))
			pamPressureVec.WithLabelValues(sensorId, parentId).Set(r.Pressure)
		}

		// only publish if there isn't a flag on the data or hardware
		// transitory events may lead to nonsense values (e.g. bug crawling over sensor)
		if r.DataFlag == 0 && !r.HwFlag {
			// publish the raw readings
			pamPm25Vec.WithLabelValues(sensorId, parentId).Set(r.Pm25_cf1)
			pamPm100Vec.WithLabelValues(sensorId, parentId).Set(r.Pm100_cf1)

			// calc the individual AQIs for the various measurements
			aqi_pm25 := sensorAQI(r.Pm25_cf1)
			aqi_pm100 := sensorAQI(r.Pm100_cf1)

			// publish the individual AQI calculations
			pamPm25AQIVec.WithLabelValues(sensorId, parentId).Set(aqi_pm25)
			pamPm100AQIVec.WithLabelValues(sensorId, parentId).Set(aqi_pm100)

			// publish the calculated AQI (max of all AQI calculations)
			aqi := math.Max(aqi_pm25, aqi_pm100)
			pamAQIVec.WithLabelValues(sensorId, parentId).Set(aqi)
		}
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
