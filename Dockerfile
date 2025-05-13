FROM golang:alpine3.21 AS golang
RUN sed -i 's#https\?://dl-cdn.alpinelinux.org/alpine#https://mirrors.tuna.tsinghua.edu.cn/alpine#g' /etc/apk/repositories
RUN apk update
RUN apk --no-cache add make git zip tzdata ca-certificates nodejs npm gcc musl-dev
WORKDIR /app
COPY . .
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go build clash-converter

FROM alpine:3.21

# Dependencies
RUN sed -i 's#https\?://dl-cdn.alpinelinux.org/alpine#https://mirrors.tuna.tsinghua.edu.cn/alpine#g' /etc/apk/repositories
RUN apk update
RUN apk --no-cache add tzdata ca-certificates

# Where application lives
WORKDIR /app
# Copy the products
COPY --from=golang /app/clash-converter .
# env
ENV GIN_MODE="release"
EXPOSE 8080

ENTRYPOINT ["/app/clash-converter"]