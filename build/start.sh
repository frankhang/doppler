#!/bin/bash

#for linux

version=1.0

if [ $# -gt 2 ]; then
  echo "Usage: $0 [service] [-r]"
  echo "  [service]: prom/grafana/doppler/client/server, server means prom + grafana + doppler. Default is blank, means start them all"
  echo "  [-r]: remove image before running"
  exit 0
fi

if [ "$1" = "" -o "$1" = "-r"  ]; then
  grafana=true
  prom=true
  doppler=true
  client=true
fi

arg=ops

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
  docker run --name grafana --rm -d --network=host --add-host=host.docker.internal:127.0.0.1 $image
fi

if [ $prom ]; then
  echo -e
  echo "#### staring prom ####"
  image=frankhang/prom-$arg:$version
  if [ $remove ]; then
    docker image rm -f $image
  fi

    docker run --name prom --rm -d --network=host --add-host=host.docker.internal:127.0.0.1 -v ~/promdata:/etc/prometheus/data $image
    #docker run --name prom --rm -d --network=host --add-host=host.docker.internal:127.0.0.1 $image

fi



if [ $doppler ]; then
  echo -e
  echo "#### staring doppler ####"
  image=frankhang/doppler:$version
  if [ $remove ]; then
    docker image rm -f $image
  fi
  docker run --name doppler --rm -d --network=host --add-host=host.docker.internal:127.0.0.1 $image -L=debug

fi

if [ $client ]; then
  echo -e
  echo "#### starting client  ####"
  image=frankhang/client:$version
  if [ $remove ]; then
    docker image rm -f $image
  fi
  docker run --name client --rm -d --network=host --add-host=host.docker.internal:127.0.0.1 $image
fi
