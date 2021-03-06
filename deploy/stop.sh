#!/bin/bash

#for linux


#!/bin/bash

#for mac

version=1.0



if [ $# = 0 ]; then
  grafana=true
  prom=true
  doppler=true
  telegraf=true
  agent=true
  client=true
fi

if [ "$1" = "server" ]; then
  grafana=true
  prom=true
  doppler=true
  telegraf=true

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

if [ "$1" = "telegraf" ]; then
  telegraf=true
fi

if [ "$1" = "agent" ]; then
  agent=true
fi

if [ "$1" = "client" ]; then
  client=true
fi

if [ "$1" = "tele" ]; then
  telegraf=true
  agent=true
fi


if [ $grafana ]; then
  echo -e
  echo "#### stoping grafana ####"
  docker stop grafana
  docker rm grafana
fi

if [ $prom ]; then
  echo -e
  echo "#### stoping prom ####"
  docker stop prom
  docker rm prom
fi

if [ $doppler ]; then
  echo -e
  echo "#### stoping doppler ####"
  docker stop doppler
  docker rm doppler

fi

if [ $telegraf ]; then
  echo -e
  echo "#### stoping telegraf ####"
  docker stop telegraf
  docker rm telegraf
fi

if [ $agent ]; then
  echo -e
  echo "#### stoping agent ####"
  docker stop agent
  docker rm agent
fi

if [ $client ]; then
  echo -e
  echo "#### stoping client  ####"
  docker stop client1
  docker rm client1
  docker stop client2
  docker rm client2
fi







