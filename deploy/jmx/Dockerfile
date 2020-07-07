FROM telegraf:1.12.6

WORKDIR /etc/telegraf

COPY jmx.toml .
ENTRYPOINT ["/usr/bin/telegraf", "--config", "/etc/telegraf/jmx.toml"]
