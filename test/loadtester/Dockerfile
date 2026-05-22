FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY main.go .
RUN CGO_ENABLED=0 go build -o /loadtester main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /loadtester /loadtester
CMD ["/loadtester"]
