package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func fetchMetrics(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("error fetching metrics: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	return string(body), nil
}

func pushMetrics(metrics string, pushGatewayUrl string) error {
	// Create temporary file with metrics content
	fileBuffer := bytes.NewBufferString(metrics)

	// Create a http request to Pushgateway
	req, err := http.NewRequest("PUT", pushGatewayUrl, fileBuffer)
	if err != nil {
		return fmt.Errorf("error creating request to Pushgateway: %v", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request to Pushgateway: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code when pushing metrics: %v", resp.Status)
	}

	log.Println("Metrics successfully pushed to Pushgateway")
	return nil
}

func main() {
	exporterUrl := "http://localhost:9101/metrics"
	pushGatewayUrl := "http://localhost:9091/metrics/job/exporter_push_job"

	// Fetch metrics from the exporter
	metrics, err := fetchMetrics(exporterUrl)
	if err != nil {
		log.Fatalf("Error fetching metrics from exporter: %v", err)
	}
	log.Println("Fetched metrics from exporter")

	// Push metrics to the Pushgateway
	if err := pushMetrics(metrics, pushGatewayUrl); err != nil {
		log.Fatalf("Error pushing metrics to Pushgateway: %v", err)
	}
}
