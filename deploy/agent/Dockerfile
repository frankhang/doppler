FROM telegraf:1.14.5

WORKDIR /etc/telegraf

COPY agent.toml .
ENTRYPOINT ["/usr/bin/telegraf", "--config", "/etc/telegraf/agent.toml"]
