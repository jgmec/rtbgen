FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
COPY *.go .
RUN go build -o rtb-generator .

FROM alpine:3.21

RUN adduser -D -u 1000 appuser

WORKDIR /app
COPY --from=builder /app/rtb-generator .

RUN mkdir -p /app/data && chown appuser /app/data

USER appuser

EXPOSE 8080

ENTRYPOINT ["./rtb-generator", "-server"]
