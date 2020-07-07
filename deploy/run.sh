#!/bin/bash

#for linux

version=1.0

if [ $# -gt 2 ]; then
  echo "Usage: $0 [service] [-r]"
  echo "  [service]: prom/grafana/doppler/telegraf/agent/client/server, server means prom + grafana + doppler. Default is blank, means start them all"
  echo "  [-r]: remove image before running"
  exit 0
fi

if [ "$1" = "" -o "$1" = "-r" ]; then
  grafana=true
  prom=true
  doppler=true
  telegraf=true
  agent=true
  client=true
fi

arg=ops

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

if [ "$1" = "-r" -o "$2" = "-r" ]; then
  remove=true
fi

if [ $grafana ]; then
  echo -e
  echo "#### staring grafana ####"
  image=frankhang/grafana:$version
  if [ $remove ]; then
    docker image rm -f $image
  fi
  docker stop grafana
  docker rm grafana
  mkdir /grafanadata
  chmod a+rwx /grafanadata
  docker run --name grafana -d --network=host --add-host=host.docker.internal:127.0.0.1 -v /grafanadata:/var/lib/grafana $image
fi

if [ $prom ]; then
  echo -e
  echo "#### staring prom ####"
  image=frankhang/prom-$arg:$version
  if [ $remove ]; then
    docker image rm -f $image
  fi
  docker stop prom
  docker rm prom

  mkdir /promdata
  chmod a+rwx /promdata
  docker run --name prom -d --network=host --add-host=host.docker.internal:127.0.0.1 -v /promdata:/etc/prometheus/data $image

fi

if [ $doppler ]; then
  echo -e
  echo "#### staring doppler ####"
  image=frankhang/doppler:$version
  if [ $remove ]; then
    docker image rm -f $image
  fi
  docker stop doppler
  docker rm doppler
  docker run --name doppler -d --network=host --add-host=host.docker.internal:127.0.0.1 $image -L=debug

fi

if [ $telegraf ]; then
  echo -e
  echo "#### starting telegraf ####"
  image=frankhang/telegraf:$version
  if [ $remove ]; then
    docker image rm -f $image
  fi
  docker stop telegraf
  docker rm telegraf
  docker run --name telegraf -d --network=host --add-host=host.docker.internal:127.0.0.1 $image
fi

if [ $agent ]; then
  echo -e
  echo "#### starting agent ####"
  image=frankhang/agent:$version
  docker stop agent
  docker rm agent
  docker run --name agent -d --network=host --add-host=host.docker.internal:127.0.0.1 $image
fi

if [ $client ]; then
  echo -e
  echo "#### starting client1  ####"
  image=frankhang/client:$version
  if [ $remove ]; then
    docker image rm -f $image
  fi
  docker stop client1
  docker rm client1
  docker stop client2
  docker rm client2
  docker run --name client1 -d --network=host --add-host=host.docker.internal:127.0.0.1 $image
  docker run --name client2 -d --network=host --add-host=host.docker.internal:127.0.0.1 $image
fi
