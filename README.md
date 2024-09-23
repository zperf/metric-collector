# metric-collector

Collect metrics from exporters and send them to [Pushgateway][Pushgateway].

## Getting started

```bash
metric-collector run --jobs node --sources http://127.0.0.1:9100/metrics \
--push-to http://127.0.0.1:9091/metrics
```

## Build from source

```bash
task         # or `task fast-build` for building
task build   # build all architectures
task docker  # build docker images 
```

[Pushgateway]: https://prometheus.io/docs/practices/pushing/ "When to use the Pushgateway"
