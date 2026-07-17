FROM golang:1.24-alpine AS builder

WORKDIR /src
ENV GOPROXY=https://goproxy.cn|https://goproxy.io|direct
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/agent-go .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/agent-go ./agent-go
EXPOSE 8080
ENTRYPOINT ["./agent-go"]
CMD ["-config", "config.json", "-listen", ":8080"]
