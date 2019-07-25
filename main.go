package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/html"
	"lib.hemtjan.st/client"
	"lib.hemtjan.st/device"
	"lib.hemtjan.st/feature"
	"lib.hemtjan.st/transport/mqtt"
)

func main() {
	flgURL := flag.String("scrape.url", "http://celsius.met.uu.se/geocelsiuswww/obs_uppsala.htm", "URL to scrape")
	flgInterval := flag.Uint("scrape.interval", 600, "Interval (in seconds) between scrapes")
	flgName := flag.String("device.name", "outside", "Device Name (in topic)")
	mqttCfg := mqtt.MustFlags(flag.String, flag.Bool)
	flag.Parse()

	if *flgInterval < 1 {
		log.Fatal("scrape.interval must be >= 1")
	}

	ctx, cancel := context.WithCancel(context.Background())

	mqttClient, err := mqtt.New(ctx, mqttCfg())

	if err != nil {
		log.Fatalf("Error starting MQTT: %v", err)
	}

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		v := false
		for {
			s, _ := <-sigCh
			if !v {
				v = true
				log.Printf("Caught signal %v, exiting", s)
				cancel()
				continue
			}
			log.Printf("Caught signal %v, terminating", s)
			os.Exit(1)
		}
	}()

	tempDev, err := client.NewDevice(&device.Info{
		Topic: "sensor/temperature/" + *flgName,
		Name:  *flgName + " Temperature",
		Type:  device.TemperatureSensor.String(),
		Features: map[string]*feature.Info{
			feature.CurrentTemperature.String(): {},
		},
	}, mqttClient)

	humidDev, err := client.NewDevice(&device.Info{
		Topic: "sensor/humidity/" + *flgName,
		Name:  *flgName + " Relative Humidity",
		Type:  device.HumiditySensor.String(),
		Features: map[string]*feature.Info{
			feature.CurrentRelativeHumidity.String(): {},
		},
	}, mqttClient)

	otherDev, err := client.NewDevice(&device.Info{
		Topic: "sensor/weather/" + *flgName,
		Name:  *flgName + " Weather",
		Type:  "weatherStation",
		Features: map[string]*feature.Info{
			"precipitation":   {},
			"airPressure":     {},
			"globalRadiation": {},
			"windSpeed":       {},
			"windDirection":   {Min: 0, Max: 360, Step: 1},
		},
	}, mqttClient)

	ticker := time.NewTicker(time.Second * time.Duration(*flgInterval))
	tick := make(chan time.Time, 2)

	go func() {
		tick <- time.Now()
		defer close(tick)
		for {
			v, o := <-ticker.C
			if !o {
				return
			}
			tick <- v
		}
	}()

	defer ticker.Stop()
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case _, open := <-tick:
			if !open {
				break loop
			}
			log.Printf("Trying to fetch %s", *flgURL)
			data, err := fetch(*flgURL)
			if err != nil {
				log.Printf("Error fetching: %s", err)
				continue
			}
			vals, err := parse(data)
			if err != nil {
				log.Printf("Error parsing: %s", err)
				continue
			}
			for _, v := range vals {

				log.Printf("%s: %s (%s)", v.Name, v.Value, v.Unit)
				switch v.Name {
				case "temperature":
					tempDev.Feature(feature.CurrentTemperature.String()).Update(v.Value)
				case "air humidity":
					humidDev.Feature(feature.CurrentRelativeHumidity.String()).Update(v.Value)
				case "precipitation last hour":
					if v.Unit == "mm (disdrometer)" {
						otherDev.Feature("precipitation").Update(v.Value)
					}
				case "wind speed":
					otherDev.Feature("windSpeed").Update(v.Value)
				case "wind direction":
					otherDev.Feature("windDirection").Update(v.Value)
				case "air pressure":
					otherDev.Feature("airPressure").Update(v.Value)
				case "global radiation":
					otherDev.Feature("globalRadiation").Update(v.Value)
				}
			}

		}
	}
}

type Value struct {
	Name  string
	Value string
	Unit  string
}

func fetch(url string) (*html.Node, error) {
	rsp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if rsp.StatusCode != 200 {
		_, _ = ioutil.ReadAll(rsp.Body)
		return nil, fmt.Errorf("http error %03d: %s", rsp.StatusCode, rsp.Status)
	}
	return html.Parse(rsp.Body)
}

func findChildren(node *html.Node, tag string) (c []*html.Node) {
	n := node.FirstChild
	for {
		if n == nil {
			return
		}
		if n.Type == html.ElementNode {
			if n.Data == tag {
				c = append(c, n)
			}
			c = append(c, findChildren(n, tag)...)
		}
		n = n.NextSibling
	}
}

func nodeText(node *html.Node) string {
	if node == nil {
		return ""
	}
	if node.Type == html.TextNode {
		return node.Data
	}
	s := ""
	if node.Type == html.ElementNode {
		n := node.FirstChild
		for {
			if n == nil {
				break
			}
			s = s + nodeText(n)
			n = n.NextSibling
		}
	}
	return s
}

func parse(data *html.Node) (vals []Value, err error) {
	if data == nil {
		return nil, errors.New("nil body")
	}

	chld := findChildren(data, "tr")

	lv := Value{}

	for _, n := range chld {
		cols := findChildren(n, "td")
		if len(cols) == 4 {
			v := Value{
				Name:  strings.ToLower(nodeText(cols[1])),
				Value: nodeText(cols[2]),
				Unit:  nodeText(cols[3]),
			}
			if len(v.Name) > 0 && len(lv.Name) > 0 && v.Name[0:1] == "-" {
				v.Name = lv.Name + " " + v.Name
			} else {
				lv = v
			}

			vals = append(vals, v)
		}
	}

	return
}
