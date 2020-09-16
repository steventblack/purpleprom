package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// Config defines the structure for the configuration information used by the application.
// It defaults to a JSON-encoded file named "purpleprom.conf" in the current working directory.
// The configuration is expressed as strict JSON, so unfortunately comments are not supported.
type Config struct {
	PollInterval Duration `json:"pollinterval"`
	Sensors      []int    `json:"sensors"`
	Metrics      Metrics  `json:"metrics"`
}

type Metrics struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path"`
	Port    int    `json:"port"`
}

// configLoad reads in the specified file and attempts to unmarshal it.
// The Config struct is
func configLoad(p string) *Config {
	f, err := os.Open(p)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer f.Close()

	// read file into a bytestream
	b, _ := ioutil.ReadAll(f)

	// create a Config and unmarshal the bytestream into it
	c := new(Config)
	err = json.Unmarshal(b, c)
	if err != nil {
		log.Fatal(err.Error())
	}

	// basic sanity checking and defaults if unspecified
	if len(c.Sensors) <= 0 {
		log.Fatal("No sensors specified in configuration.")
	}
	if c.PollInterval <= 0 {
		c.PollInterval, _ = parseDuration("60s")
	}

	return c
}

// UnmarshalJSON provides an interface for customized processing of the Metrics element.
// It performs the initialization of select fields to default values prior to the actual unmarshaling.
// The default values will be overwritten if present in the config.
func (m *Metrics) UnmarshalJSON(data []byte) error {
	m.Enabled = true
	m.Port = 6005
	m.Path = "/metrics"

	// avoid circular reference
	type Alias Metrics
	tmp := (*Alias)(m)

	return json.Unmarshal(data, tmp)
}

// The Duration type provides enables the JSON module to process strings as time.Durations.
// While time.Duration is available as a native type for CLI flags, it is not for the JSON parser.
// Note that in Go, you cannot define new methods on a non-local type so this workaround is the
// best alternative to hacking directly in the standard Go time module.
type Duration time.Duration

// Duration returns the time.Duration native type of the time module.
// This helper function makes it slightly less tedious to continually typecast a Duration into a time.Duration
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// ParseDuration is a helper function to parse a string utilizing the underlying time.ParseDuration functionality.
func parseDuration(s string) (Duration, error) {
	td, err := time.ParseDuration(s)
	if err != nil {
		return Duration(0), err
	}

	return Duration(td), nil
}

// MarshalJSON supplies an interface for processing Duration values which wrap the standard time.Duration type.
// It returns a byte array and any error encountered.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON supplies an interface for processing Duration values which wrap the standard time.Duration type.
// It accepts a byte array and returns any error encountered.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return fmt.Errorf("Invalid Duration specification: '%v'", value)
	}
}
