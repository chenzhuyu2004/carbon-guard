# -------- Build stage --------
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o carbon-guard

# -------- Runtime stage --------
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/carbon-guard .
COPY entrypoint.sh .

RUN chmod +x entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
