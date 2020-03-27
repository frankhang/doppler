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
  docker stop doppler
  docker stop grafana
  docker stop prom
fi

if [ $client ]; then
  docker stop client
fi



