FROM golang:1.21-alpine AS builder
RUN apk add --no-cache make git gcc musl-dev
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o bin/idea-arena-server ./cmd/server/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /app/bin/idea-arena-server .
COPY --from=builder /app/configs ./configs
RUN mkdir -p /app/data /app/logs
EXPOSE 8000
CMD ["./idea-arena-server", "-conf", "./configs/"]
