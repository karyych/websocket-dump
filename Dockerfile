# 1) билд
FROM golang:1.21-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod verify && \
    go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ws-server ./server.go

# 2) рантайм
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /app/ws-server /app/ws-server
COPY static /app/static
EXPOSE 8765
USER nonroot:nonroot
ENTRYPOINT ["/app/ws-server"]
