# Metric collector

Collect metrics from exporters and send them to [Pushgateway][Pushgateway].

## Getting started

```bash
metric-collector run --jobs node \
--sources http://127.0.0.1:9100/metrics \
--targets http://127.0.0.1:9091/metrics
```

[Pushgateway]: https://prometheus.io/docs/practices/pushing/ "When to use the Pushgateway"
