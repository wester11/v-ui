FROM golang:1.22-alpine AS build
WORKDIR /src
# Copy everything first — go mod tidy needs source to resolve imports
COPY . .
# go mod tidy generates go.sum if missing, downloads all deps
RUN go mod tidy
# Build the agent binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/agent ./cmd/agent

FROM alpine:3.19
# docker-cli — для xray-режима: агент рестартует sidecar через `docker restart`
# (Phase 5.1 stateless модель). Для WG/AWG-режимов docker-cli не используется.
RUN apk add --no-cache wireguard-tools iproute2 iptables docker-cli
COPY --from=build /out/agent /agent
EXPOSE 7443/tcp 51821/udp
ENTRYPOINT ["/agent"]
