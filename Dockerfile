FROM golang:1.21.4-alpine AS builder
LABEL stage=gobuilder \
      mainatiner=https://github.com/mayooot/gpu-docker-api

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
RUN apk add gcc g++ make libffi-dev openssl-dev libtool

ENV CGO_ENABLED 0
ENV GOPROXY https://goproxy.cn,direct

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN make linux_no_ldflags

FROM nvidia/cuda:10.0-base

VOLUME /data
WORKDIR /data

COPY --from=builder /build/bin/gpu-docker-api-linux-amd64 /data/gpu-docker-api

EXPOSE 2378

CMD ["/data/gpu-docker-api"]