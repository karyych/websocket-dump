# 1) билд
FROM golang:1.20 AS build

# Установка рабочей директории
WORKDIR /app

# Копирование и проверка зависимостей
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Копирование исходного кода
COPY . .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ws-server ./server.go

# 2) рантайм
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /app/ws-server /app/ws-server
COPY static /app/static
EXPOSE 8765
USER nonroot:nonroot
ENTRYPOINT ["/app/ws-server"]
