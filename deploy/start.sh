#!/bin/bash
docker run --name prom --rm -d -p 9090:9090 frankhang/prom-ops:1.0
docker run --name grafana --rm -d -p 4000:4000 frankhang/grafana:1.0
docker run --name doppler --rm -d -p 8125:8125/udp -p 8825:8825 frankhang/doppler:1.0 -L=debug