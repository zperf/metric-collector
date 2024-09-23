package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/MisterKaiou/go-functional/result"
	"github.com/adrianbrad/queue"
	"github.com/cockroachdb/errors"
	"github.com/go-resty/resty/v2"
	"github.com/rogpeppe/generic/tuple"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var errEndOfQueue = errors.New("end of queue")

func startPushWorker(ctx context.Context, metricsQueue MetricQueue, config pushConfig) {
	client := resty.New().SetTimeout(config.pushTimeout)

	go func() {
	pushLoop:
		for {
			select {
			case <-ctx.Done():
				break pushLoop
			default:
			}

			t := metricsQueue.GetWait()
			sourceIndex := t.A0
			r := t.A1
			if r.IsError() {
				err := r.UnwrapError()
				if errors.Is(err, errEndOfQueue) {
					break pushLoop
				}
				log.Error().Err(err).Msg("Consume queue failed")
				continue
			}

			job := config.jobs[sourceIndex]
			instance := config.instance
			url := fmt.Sprintf("%s/job/%s/instance/%s", config.pushTo, job, instance)
			body := r.Unwrap()
			rsp, err := client.R().SetBody(body).Put(url)
			if err != nil {
				log.Error().Err(err).Str("job", job).Str("instance", instance).Msg("Push failed")
			} else if rsp.StatusCode() != http.StatusOK {
				log.Error().Int("code", rsp.StatusCode()).Str("body", rsp.String()).Msg("Push failed")
			}
		}

		log.Info().Msg("Push loop exits")
	}()
}

func collectMetric(client *resty.Client, source string) result.Of[string] {
	rsp, err := client.R().Get(source)
	if err != nil {
		return result.Error[string](err)
	}
	body := string(rsp.Body())
	return result.Ok(body)
}

type MetricQueue = *queue.Blocking[tuple.T2[int, result.Of[string]]]

func startCollectWorkers(ctx context.Context, metricsQueue MetricQueue, config collectConfig) {
	ticker := time.NewTicker(config.collectInterval)
	client := resty.New().SetTimeout(config.collectTimeout)

collectLoop:
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			break collectLoop

		case <-ticker.C:
			var wg sync.WaitGroup
			wg.Add(len(config.metricsSources))

			results := make([]tuple.T2[int, result.Of[string]], len(config.metricsSources))
			for i, source := range config.metricsSources {
				go func(sourceIndex int) {
					results[sourceIndex] = tuple.MkT2(sourceIndex, collectMetric(client, source))
					wg.Done()
				}(i)
			}
			wg.Wait()

			for _, t := range results {
				sourceIndex := t.A0
				r := t.A1
				if r.IsError() {
					log.Error().Err(r.UnwrapError()).Int("source-index", sourceIndex).Msg("Collect failed")
				} else {
					metricsQueue.OfferWait(t)
				}
			}
		}
	}
}

var CmdRun = &cli.Command{
	Name:  "run",
	Usage: "Run push exporter server",
	Flags: []cli.Flag{
		flagMetricSources,
		flagMetricJobs,
		flagCollectInterval,
		flagCollectTimeout,
		flagPushTo,
		flagPushTimeout,
		flagInstance,
		flagMetricQueueDepth,
	},
	Action: func(c *cli.Context) error {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		q := queue.NewBlocking[tuple.T2[int, result.Of[string]]]([]tuple.T2[int, result.Of[string]]{}, queue.WithCapacity(c.Int("queue-depth")))
		startPushWorker(ctx, q, newPushConfig(c))
		startCollectWorkers(ctx, q, newCollectConfig(c))

		<-ctx.Done()
		return nil
	},
}
