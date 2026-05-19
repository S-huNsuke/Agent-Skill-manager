# 此 Dockerfile 仅用于开发环境和 CI 构建，不用于生产部署。
# 生产环境请使用官方发布渠道获取预编译二进制文件。
FROM golang:1.26-bookworm AS builder

RUN apt-get update && apt-get install -y \
    build-essential \
    libgtk-3-dev \
    libwebkit2gtk-4.1-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN cd frontend && \
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install -y nodejs && \
    npm install -g pnpm && \
    pnpm install && \
    pnpm build && \
    cd ..

RUN CGO_ENABLED=1 go build \
    -tags "desktop,production" \
    -ldflags "-w -s" \
    -o /app/build/bin/agent-skills-manager \
    ./cmd/agent-skills-manager/

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    libgtk-3-0 \
    libwebkit2gtk-4.1-0 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/build/bin/agent-skills-manager /app/agent-skills-manager

COPY --from=builder /app/build/appicon.icns /app/build/appicon.icns

RUN mkdir -p /app/data

ENV ASM_DATA_DIR=/app/data

EXPOSE 8080

ENTRYPOINT ["/app/agent-skills-manager"]
