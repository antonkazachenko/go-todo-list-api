FROM golang:1.22.2 as builder

WORKDIR /app

COPY . .

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc libc6-dev \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

RUN go mod download

RUN CGO_ENABLED=1 GOOS=linux go build -o main .

FROM ubuntu:latest

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/web ./web

RUN chmod +x /app/main

ENV TODO_PORT=7540
ENV TODO_DBFILE=scheduler.db
ENV TODO_PASSWORD=test12345

EXPOSE 7540

CMD ["/app/main"]
