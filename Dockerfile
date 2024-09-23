FROM fedora:40

COPY metric-collector /usr/local/bin/
RUN chmod +x /usr/local/bin/metric-collector

ENTRYPOINT ["/usr/local/bin/metric-collector"]
