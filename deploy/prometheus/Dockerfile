FROM prom/prometheus:v2.17.1

WORKDIR /etc/prometheus/

ARG config
COPY ${config}.* ./
RUN mv ${config}.yml prometheus.yml

CMD [ "--web.listen-address=:8090" ]
ENTRYPOINT ["/bin/prometheus", "--config.file=/etc/prometheus/prometheus.yml", "--web.console.libraries=/usr/share/prometheus/console_libraries", "--web.console.templates=/usr/share/prometheus/consoles"]


