FROM golang:alpine AS builder
WORKDIR /app
COPY . .
ARG GITHUB_SHA
ARG VERSION
# 使用多代理回落,避免单一 proxy 波动导致构建失败
ENV GOPROXY=https://proxy.golang.org,https://goproxy.cn,direct
RUN echo "Building commit: ${GITHUB_SHA:0:7}" && \
    go mod download && \
    go build -ldflags="-s -w -X main.Version=${VERSION} -X main.CurrentCommit=${GITHUB_SHA:0:7}" -trimpath -o subs-check .

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
