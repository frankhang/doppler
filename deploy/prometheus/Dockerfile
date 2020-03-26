FROM prom/prometheus:latest AS builder

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /etc/prometheus/console_libraries ./console_libraries
COPY --from=builder /etc/prometheus/consoles ./consoles
COPY --from=builder /bin/prometheus .
RUN mkdir data

ARG config
COPY ${config}.* ./
RUN mv ${config}.yml prometheus.yml
# COPY ${config}.yml ./prometheus.yml
ENTRYPOINT ["./prometheus", "--config.file=prometheus.yml"]