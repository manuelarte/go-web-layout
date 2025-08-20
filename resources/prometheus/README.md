# Prometheus

You can run Prometheus locally using:

```bash
docker run \
    --add-host host.docker.internal=host-gateway \
    -p 9090:9090 \
    -v ./prometheus.yml:/etc/prometheus/prometheus.yml \
    prom/prometheus
```
