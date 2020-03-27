#!/bin/bash

#for linux

if [ $# = 0 ]; then
    server=true
    client=true
fi

if [ "$1" = "server"  ]; then
    server=true

fi

if [ "$1" = "client"  ]; then
    client=true
fi


if [ $server ]; then
  docker run --name prom --rm -d --network=host --add-host=host.docker.internal:127.0.0.1 frankhang/prom-ops:1.0
  docker run --name grafana --rm -d --network=host --add-host=host.docker.internal:127.0.0.1 frankhang/grafana:1.0
  docker run --name doppler --rm -d --network=host --add-host=host.docker.internal:127.0.0.1 frankhang/doppler:1.0
fi

if [ $client ]; then
  docker run --name client --rm -d --network=host --add-host=host.docker.internal:127.0.0.1 frankhang/client:1.0
fi
