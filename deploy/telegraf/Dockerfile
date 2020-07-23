FROM telegraf:1.14.5

WORKDIR /etc/telegraf

COPY telegraf.toml .
ENTRYPOINT ["/usr/bin/telegraf", "--config", "/etc/telegraf/telegraf.toml"]
