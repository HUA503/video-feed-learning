# 第一阶段：编译 Go 程序
FROM golang:1.27.1 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/video-feed .


# 第二阶段：准备运行环境
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /out/video-feed /app/video-feed
COPY --from=builder /src/web /app/web

USER 10001:10001

EXPOSE 8080

STOPSIGNAL SIGINT

ENTRYPOINT ["/app/video-feed"]
