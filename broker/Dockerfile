FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/broker ./cmd/server

FROM alpine:3.21

WORKDIR /app

COPY --from=build /out/broker /app/broker
COPY migrations /app/migrations

EXPOSE 8080

CMD ["/app/broker"]
