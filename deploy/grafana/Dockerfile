FROM frankhang/grafana:4.0 AS builder

FROM grafana/grafana
ENV GOPROXY="https://goproxy.cn,direct"
ENV PRJ_DIR="github.com/grafana/"

WORKDIR /usr/share/grafana
COPY --from=builder /go/src/${PRJ_DIR}/dist/grafana/bin/ bin
#COPY --from=builder /go/src/${PRJ_DIR}/dist/grafana/bin/grafana-server bin/grafana-server

CMD ["bin/grafana-server", "web"]