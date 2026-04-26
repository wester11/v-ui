FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/agent ./cmd/agent

FROM alpine:3.19
RUN apk add --no-cache wireguard-tools iproute2 iptables
COPY --from=build /out/agent /agent
EXPOSE 7443/tcp 51821/udp
ENTRYPOINT ["/agent"]
