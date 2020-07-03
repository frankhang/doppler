FROM telegraf:1.12.6

WORKDIR /etc/telegraf

COPY telegraf.toml .
ENTRYPOINT ["/usr/bin/telegraf", "--config", "/etc/telegraf/telegraf.toml"]
