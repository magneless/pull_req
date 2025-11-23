FROM golang:1.25 AS build

RUN go install github.com/codesenberg/bombardier@latest

WORKDIR /app

COPY seeder ./seeder
COPY entrypoint.sh .

RUN chmod +x entrypoint.sh

ENTRYPOINT ["./entrypoint.sh"]