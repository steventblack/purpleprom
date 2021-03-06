# PurpleProm
A reader of PurpleAir AQI devices which exports the data for Prometheus consumption

## Setup
This application is built using Go and should run on any system supported by that language. It can be built using the standard ```make``` utility:
```bash
make build
```
This will build an executable suitable for the local system and place it in a newly-created **purpleprom/bin** directory.

## Configuration
Sensor IDs can be identified by selecting the sensor(s) in question on the map provided at [PurpleAir](https://www.purpleair.com/map). Click on a sensor and note the value for the _select_ attribute in the URL. 

> e.g. In the URL https://www.purpleair.com/map?opt=1/mAQI/a10/cC0&select=37011#15.94/37.437227/-122.198933, the sensor ID is 37011.

The configuration file must be named ```purpleprom.conf``` and be located in the current working directory. The config file itself is in JSON format and provides some degree of customization.

```
{
  "pollinterval": "60s",           // optional: default is 60s
  "sensors": [ 12345, 67890 ],     // required: at least one sensorID must be provided
  "metrics": {                     // optional: entire block may be omitted if defaults acceptable
    "enabled": true,               // optional: default is true
    "path": "/metrics",            // optional: default is "/metrics"
    "port": 6005                   // optional: default is 6005
  }
}
```

  * The ```pollinterval``` attribute will accept any number of time formats that can be parsed by the [time.ParseDuration](https://godoc.org/time#ParseDuration) function in Go. It is strongly recommended not to poll more frequently than once a minute in order to avoid getting rate-limited by PurpleAir. In general, 5-15 minute poll intervals should provide reasonable coverage of changing air quality or weather conditions. 
  * The ```sensors``` attribute allows for multiple sensors to be named and it may be desirable to collect information from more than one sensor in the area in order to detect any discrepencies in the readings. e.g. A spike in particulate whenever a car drives nearby or an improperly placed temperature gauge. 
  * The ```metrics``` object (and all of its attributes) is _optional_ and may be omitted entirely. 
    * ```enabled``` is a boolean enabling/disabling the metrics output.
    * ```path``` is the path the metrics will be published for Prometheus scraping. 
    * ```port``` is the listening port for the metrics server.

## Prometheus Setup
The **prometheus.yml** file needs to be extended to include scraping of the purpleprom service. Customizing the following block should be all that is required:

```
- job_name: 'purpleair'                        // the job name can be whatever you like
  scrape_interval: 30s                         // this should be 1/2 of the pollinterval time
  scrape_timeout: 10s                          // should be shorter than the scrape_interval
  static_configs:
    - targets: ['hostname.example.com:6005']   // full hostname and port where purpleprom is running
```

If the ```path``` attribute in **purpleprom.conf** has been changed from the default "/metrics", then the **prometheus.yml** file needs to include a ```metrics_path``` attribute with a matching path. If there is a firewall between the Prometheus scraper and the purpleprom app, then a firewall hole must be opened for the destination and port.

## Running
There are several options that can be specified on the command-line when running the application:
```bash
purpleprom [-c config_path] [-o logging_path]
```
  * **[-c path]** is optional and specifies the path the application should use for its configuration file. If not specified, the default path value of **purpleprom.conf** in the current working directory is assumed.
  * **[-o path]** is optional and specifies the output file for any logging or diagnostic information the application may produce. If not specified, the default behavior is to disable logging and run silently.
  
Note that at least one sensor _must_ be specified in the configuration file for the application to function. 

The application is easily added to the crontab or setup under systemd for automated startup/shutdown/restart handling.

manual:
> ```nohup /usr/local/bin/purpleprom -c /etc/purpleprom.conf &```

crontab:
> ```@reboot /usr/local/bin/purpleprom -c /etc/purpleprom.conf```
