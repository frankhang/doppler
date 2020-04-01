#!/bin/bash

#for linux


#!/bin/bash

#for mac

version=1.0



if [ $# = 0 ]; then
  grafana=true
  prom=true
  doppler=true
  client=true
fi

if [ "$1" = "server" ]; then
  grafana=true
  prom=true
  doppler=true
fi

if [ "$1" = "grafana" ]; then
  grafana=true
fi

if [ "$1" = "prom" ]; then
  prom=true
fi

if [ "$1" = "doppler" ]; then
  doppler=true
fi

if [ "$1" = "client" ]; then
  client=true
fi



if [ $grafana ]; then
  echo -e
  echo "#### stoping grafana ####"
  docker stop grafana
fi

if [ $prom ]; then
  echo -e
  echo "#### stoping prom ####"
  docker stop prom
fi

if [ $doppler ]; then
  echo -e
  echo "#### stoping doppler ####"
  docker stop doppler

fi

if [ $client ]; then
  echo -e
  echo "#### stoping client  ####"
  docker stop client
fi





