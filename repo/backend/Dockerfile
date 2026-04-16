FROM golang:1.25-alpine AS build
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -mod=vendor -o server ./cmd/server

FROM alpine:3.19
COPY --from=build /app/server /usr/local/bin/server
COPY --from=build /app/migrations /app/migrations
RUN mkdir -p /app/uploads /var/log/ledgermint
ENTRYPOINT ["server"]
