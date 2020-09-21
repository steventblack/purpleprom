package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
)

type paSensor struct {
	MapVersion       string           `json:"mapVersion"`
	BaseVersion      string           `json:"baseVersion"`
	MapVersionString string           `json:"mapVersionString"`
	Results          []paSensorResult `json:"results"`
}

// PurpleAir API decoded (with notes on different versions) and AQI calculation
// https://docs.google.com/document/d/15ijz94dXJ-YAZLi9iZ_RaBwrZ4KtYeCy08goGBwnbCU/edit

type paSensorResult struct {
	Id        int     `json:"ID"`
	ParentId  int     `json:"ParentID"`
	Label     string  `json:"Label"`
	Lat       float64 `json:"Lat"`
	Lon       float64 `json:"Lon"`
	DataFlag  int     `json:"Flag"`
	HwFlag    int     `json:"A_H"`
	P03um     float64 `json:"p_0_3_um,string"`
	P05um     float64 `json:"p_0_5_um,string"`
	P10um     float64 `json:"p_1_0_um,string"`
	P25um     float64 `json:"p_2_5_um,string"`
	P50um     float64 `json:"p_5_0_um,string"`
	P100um    float64 `json:"p_10_0_um,string"`
	Pm10_cf1  float64 `json:"pm1_0_cf_1,string"`
	Pm25_cf1  float64 `json:"pm2_5_cf_1,string"`
	Pm100_cf1 float64 `json:"pm10_0_cf_1,string"`
	Pm10_atm  float64 `json:"pm1_0_atm,string"`
	Pm25_atm  float64 `json:"pm2_5_atm,string"`
	Pm100_atm float64 `json:"pm10_0_atm,string"`
	Humidity  int     `json:"humidity,string"`
	Temp      int     `json:"temp_f,string"`
	Pressure  float64 `json:"pressure,string"`
	Version   string  `json:"Version"`
}

// readSensor fetches the JSON reading for the named sensorId and attempts to process it into a paSensor struct.
// The sensorId may be found by examing the "Fire and Smoke Map" at "https://fire.airnow.gov" and locating
// the sensor(s) of interest on the map. The sensor ID can be extracted by clicking on the sensor icon and examining
// the "Site ID" field: the sensor will be the digits after the last underscore character.
// e.g. for "Site ID: PA_6b88dd5af19b42b5_37011" the sensor ID is 37011. Be sure the sensor's provider is PurpleAir.
// If successful, a pointer to a paSensor struct will be returned else an error.
func sensorRead(sensorIds []int) (*paSensor, error) {
	if len(sensorIds) <= 0 {
		return nil, fmt.Errorf("No sensors specified")
	}

	// API support multiple sensorIDs on a single call separated by a "|".
	// TODO: put in a limit to prevent gross abuse
	url := "https://www.purpleair.com/json?show="
	for i, v := range sensorIds {
		if i != 0 {
			url += "|"
		}
		url += strconv.Itoa(v)
	}

	r, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status reading sensor: status code %v", r.StatusCode)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	s := new(paSensor)
	err = json.Unmarshal(body, s)
	if err != nil {
		return nil, err
	}

	// if ParentId remains the default value, update it with the Id to simplify other areas of code.
	for i, r := range s.Results {
		if r.ParentId == 0 {
			s.Results[i].ParentId = s.Results[i].Id
		}
	}

	return s, nil
}

// sensorAQI sets up the equation for calculating the Air Quality Index depending on the
// particulate matter reading. The calculation inputs vary with the pm value.
// If an invalid (negative) pm value is passed in, 0 will be returned.
func sensorAQI(pm float64) float64 {
	switch {
	case pm > 350.5:
		return sensorCalcAQI(pm, 500.0, 401.0, 500.0, 350.5)
	case pm > 250.5:
		return sensorCalcAQI(pm, 400.0, 301.0, 350.4, 250.5)
	case pm > 150.5:
		return sensorCalcAQI(pm, 300.0, 201.0, 250.4, 150.5)
	case pm > 55.5:
		return sensorCalcAQI(pm, 200.0, 151.0, 150.4, 55.5)
	case pm > 35.5:
		return sensorCalcAQI(pm, 150.0, 101.0, 55.4, 35.5)
	case pm > 12.1:
		return sensorCalcAQI(pm, 100.0, 51.0, 35.4, 12.1)
	case pm >= 0.0:
		return sensorCalcAQI(pm, 50.0, 0.0, 12.0, 0.0)
	default:
		log.Printf("Unable to calculate AQI on invalid sensor value: %f", pm)
		return 0.0
	}
}

// sensorCalcAQI formula is based on the US EPA formula. The formula inputs vary
// depending on various "breakpoints". The official AQI is based on the highest level
// of a number of different inputs, including particle pollution (2.5um and 10um)
// and gas concentrations (Sulfur Dioxide, Nitrogen Dioxide, etc.)
// The resulting AQI should fall within 6 bands:
// 0-50 "Good"
// 51-100 "Moderate"
// 101-150 "Unhealthy for sensitive groups"
// 151-200 "Unhealthy"
// 201-300 "Very unhealthy"
// 301+ "Hazardous"
// Full details can be found at "https://www.airnow.gov/sites/default/files/2020-05/aqi-technical-assistance-document-sept2018.pdf"
// Note that the official AQI formulas call for very specific truncation at selected number of decimal places.
// Ozone: trunc@3 decimals, pm2_5 trunc@1 decimal, pm10_0 trunch@int, CO trunc@1 decimal, SO2 trunch@int, NO2 trunc@int
func sensorCalcAQI(Cp, Ih, Il, Bph, Bpl float64) float64 {
	a := Ih - Il
	b := Bph - Bpl
	c := Cp - Bpl

	aqi := math.Round((a/b)*c + Il)

	return aqi
}
