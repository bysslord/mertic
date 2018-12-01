package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/mqtt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"time"
)

const (
	PUBTOPIC = "/metric/%s"
	CMDTOPIC = "/cmd/%s"
	APPNAME  = "Observer-%s"
)

var (
	FLAG struct {
		// 配置文件
		c string
	}
	YAML struct {
		DURATION struct {
			WRITE int64
		}
		MQTT struct {
			HOST     string
			USERNAME string
			PASSWORD string
		}
	}
)

type KV map[string]interface{}

type Metric struct {
	Fields KV `json:"fields"`
	Tags   KV `json:"tags"`
}

func (metric Metric) toByteArray() []byte {
	b, err := json.Marshal([]KV{metric.Fields, metric.Tags})
	if err != nil {
		log.Printf("Error: [%s]", err.Error())
	}
	return b
}

func getMetric() []byte {

	vm, _ := mem.VirtualMemory()
	c, _ := cpu.Percent(10*time.Second, false)

	metric := Metric{
		Fields: KV{
			"cpu_percent": c,
			"mem_percent": vm.UsedPercent,
		},
		Tags: KV{},
	}
	return metric.toByteArray()
}

func init() {
	flag.StringVar(&FLAG.c, "c", "app.yaml", "Specified a config file")
	flag.Parse()

	if dat, err := ioutil.ReadFile(FLAG.c); err != nil {
		log.Fatal(err.Error())
	} else {
		err := yaml.Unmarshal(dat, &YAML)
		if err != nil {
			log.Fatalf("Parse file [%s] failed: [%s]", FLAG.c, err)
		}
	}

}

func main() {

	HOSTNAME, err := os.Hostname()
	if err != nil {
		log.Fatalf("Cannot get hostname: %s", err)
	}

	pubTopic := fmt.Sprintf(PUBTOPIC, HOSTNAME)
	subTopic := fmt.Sprintf(CMDTOPIC, HOSTNAME)
	appName := fmt.Sprintf(APPNAME, HOSTNAME)

	mqttAdaptor := mqtt.NewAdaptorWithAuth(YAML.MQTT.HOST, appName, YAML.MQTT.USERNAME, YAML.MQTT.PASSWORD)

	work := func() {
		mqttAdaptor.On(subTopic, func(msg mqtt.Message) {
			fmt.Println(msg)
		})
		gobot.Every(10*time.Second, func() {
			mqttAdaptor.Publish(pubTopic, getMetric())
		})
	}

	robot := gobot.NewRobot("mqttBot",
		[]gobot.Connection{mqttAdaptor},
		work,
	)

	robot.Start()

	recover()
}
