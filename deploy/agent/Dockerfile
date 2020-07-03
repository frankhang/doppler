FROM telegraf:1.12.6

WORKDIR /etc/telegraf

COPY agent.toml .
ENTRYPOINT ["/usr/bin/telegraf", "--config", "/etc/telegraf/agent.toml"]
