FROM golang:1.22-alpine

WORKDIR /app

COPY . .

RUN go build -o carbon-guard

ENTRYPOINT ["/app/entrypoint.sh"]
