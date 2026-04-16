FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server

FROM alpine:3.19
COPY --from=build /app/server /usr/local/bin/server
COPY --from=build /app/migrations /app/migrations
RUN mkdir -p /app/uploads /var/log/ledgermint
ENTRYPOINT ["server"]
