FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY main.go .
RUN CGO_ENABLED=0 go build -o /fake-backend main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /fake-backend /fake-backend
CMD ["/fake-backend"]
