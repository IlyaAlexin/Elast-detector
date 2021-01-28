package main

import (
	"elast-detector/kafka"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	ElasticServer string        `yaml:"es_server"`
	ElasticScheme string        `yaml:"es_scheme"`
	TimeRange     time.Duration `yaml:"time_range"`
	Frequency     time.Duration `yaml:"query_frequency"`
	Rules         []Rule        `yaml:"rules"`
	Outputs       []Output      `yaml:"outputs"`
}

type Output struct {
	Name   string            `yaml:"name"`
	Type   string            `yaml:"type"`
	Otions map[string]string `yaml:"options"`
}

func processError(err error) {
	fmt.Println(err)
	os.Exit(2)
}

func readFile(config *Config, cfgFile string) {
	file, err := os.Open(cfgFile)
	if err != nil {
		processError(err)
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
		processError(err)
	}
}

func main() {
	var config Config
	var cfgFile string
	flag.StringVar(&cfgFile, "file", "config.yaml", "Application config file")
	readFile(&config, cfgFile)
	fmt.Printf("Final Config %v\n", config)
	esURL := fmt.Sprintf("%s://%s", config.ElasticScheme, config.ElasticServer)
	es, err := NewEsClient(esURL)
	if err != nil {
		processError(err)
	}
	kafkaTopic := "events"
	kafkaAddr := "random"
	kafkaConnector := kafka.Connect(kafkaAddr, kafkaTopic)
	// sched := scheduler.GetScheduler()
	// ticker := time.NewTicker(5 * time.Second)
	// done := make(chan bool)

	for _, rule := range config.Rules {
		tick := time.NewTicker(config.Frequency)
		go func() {
			for val := range tick.C {
				countFunctions, _ := es.ConstructSearch(rule)
				response, _ := es.Search(countFunctions)
				events, err := rule.GenerateEvents(response)
				if err != nil {
					fmt.Println("Failed to generate events", err.Error())
					continue
				}
				for _, event := range events {
					jsonEvent, err := json.Marshal(event)
					if err != nil {
						fmt.Println("Failed to marshal event", event)
						continue
					}
					kafkaConnector.ProduceEvent(rule.Name, jsonEvent)
				}
				fmt.Println(val, response["hits"].(map[string]interface{})["total"])
			}
		}()
		if err != nil {
			fmt.Errorf(err.Error())
		}
	}
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	})
	http.ListenAndServe(":8080", nil)
}
