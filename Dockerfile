FROM golang:1.19 AS builder

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o manager

FROM openjdk:8

RUN apt-get update && \
    apt-get install -y curl && \
    mkdir -p /usr/share/service/versions /var/log/app

COPY --from=builder /app/manager /usr/local/bin/
COPY templates /app/templates
COPY static /app/static

ENV API_KEY=your-secure-key
EXPOSE 8080

CMD ["manager"]