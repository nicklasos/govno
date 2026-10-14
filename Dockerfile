FROM golang:1.25-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/govno ./cmd/govno

FROM alpine:3.24
RUN apk add --no-cache ca-certificates mariadb-client \
    && (command -v mysqldump || ln -s "$(command -v mariadb-dump)" /usr/local/bin/mysqldump) \
    && adduser -D -u 10001 govno
COPY --from=build /out/govno /usr/local/bin/govno
USER govno
ENTRYPOINT ["govno"]
