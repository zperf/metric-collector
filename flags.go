package main

import (
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

var flagMetricQueueDepth = &cli.IntFlag{
	Name:  "queue-depth",
	Value: 128,
}

var flagInstance = &cli.GenericFlag{
	Name:  "instance",
	Value: FlagHostname{""},
}

type FlagHostname struct {
	Hostname string
}

func (f FlagHostname) Set(value string) error {
	f.Hostname = value
	return nil
}

func (f FlagHostname) String() string {
	if f.Hostname == "" {
		h, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		f.Hostname = h
	}
	return f.Hostname
}

var flagMetricSources = &cli.StringSliceFlag{
	Name:     "sources",
	Required: true,
}

var flagMetricJobs = &cli.StringSliceFlag{
	Name:     "jobs",
	Required: true,
}

var flagPushTo = &cli.StringFlag{
	Name:     "push-to",
	Required: true,
	Value:    "http://192.168.24.6:9091/metrics",
}

var flagCollectInterval = &cli.DurationFlag{
	Name:  "collect-interval",
	Value: 10 * time.Second,
}

var flagCollectTimeout = &cli.DurationFlag{
	Name:  "collect-timeout",
	Value: 9 * time.Second,
}

var flagPushTimeout = &cli.DurationFlag{
	Name:  "push-timeout",
	Value: 3 * time.Second,
}

type collectConfig struct {
	collectInterval time.Duration
	collectTimeout  time.Duration
	metricsSources  []string
}

func newCollectConfig(c *cli.Context) collectConfig {
	return collectConfig{
		collectInterval: c.Duration("collect-interval"),
		collectTimeout:  c.Duration("collect-timeout"),
		metricsSources:  c.StringSlice("sources"),
	}
}

type pushConfig struct {
	pushTimeout time.Duration
	jobs        []string
	instance    string
	pushTo      string
}

func newPushConfig(c *cli.Context) pushConfig {
	return pushConfig{
		pushTimeout: c.Duration("push-timeout"),
		jobs:        c.StringSlice("jobs"),
		instance:    c.String("instance"),
		pushTo:      c.String("push-to"),
	}
}
