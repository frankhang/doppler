FROM grafana/grafana

WORKDIR /usr/share/grafana

COPY defaults.ini conf/
ENTRYPOINT ["bin/grafana-server", "web"]