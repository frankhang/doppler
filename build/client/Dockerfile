FROM golang:1.14.1 AS builder
ENV GOPROXY="https://goproxy.cn,direct"
ENV PRJ_DIR="github.com/frankhang/doppler/"
ENV APP_DIR="mock/client/"
ENV APP_NAME="client"
WORKDIR /go/src/${PRJ_DIR}
COPY . .
WORKDIR /go/src/${PRJ_DIR}${APP_DIR}
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ${APP_NAME} .

FROM alpine:latest
#FROM ubuntu
ENV PRJ_DIR="github.com/frankhang/doppler/"
ENV APP_DIR="mock/client/"
ENV APP_NAME="client"
WORKDIR /root/
COPY --from=builder /go/src/${PRJ_DIR}${APP_DIR}${APP_NAME} .
ADD https://raw.githubusercontent.com/frankhang/doppler/master/build/doppler/doppler.toml .
ENTRYPOINT ["./client"]