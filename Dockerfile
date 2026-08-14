FROM golang:alpine AS builder
WORKDIR /app
COPY . .
ARG GITHUB_SHA
ARG VERSION
# 依赖已 vendoring(vendor/ 目录入库),使用 -mod=vendor 离线构建,无需访问任何 Go proxy,
# 彻底避免构建时 proxy.golang.org 不稳定导致 go mod download 失败
ENV GOFLAGS=-mod=vendor GOPROXY=off CGO_ENABLED=0
RUN echo "Building commit: ${GITHUB_SHA:0:7}" && \
    go build -mod=vendor -ldflags="-s -w -X main.Version=${VERSION} -X main.CurrentCommit=${GITHUB_SHA:0:7}" -trimpath -o subs-check .

FROM alpine
WORKDIR /app
ENV TZ=Asia/Shanghai
RUN apk add --no-cache alpine-conf ca-certificates && \
    /usr/sbin/setup-timezone -z Asia/Shanghai && \
    apk del alpine-conf && \
    rm -rf /var/cache/apk/*
COPY --from=builder /app/subs-check /app/subs-check
CMD ["/app/subs-check"]
EXPOSE 8199
