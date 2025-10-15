# 1) билд
FROM golang:1.24.5-alpine AS build

# Установка необходимых инструментов для сборки
RUN apk add --no-cache git ca-certificates tzdata

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
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /app/ws-server /app/ws-server
COPY static /app/static
EXPOSE 8765
USER nonroot:nonroot
ENTRYPOINT ["/app/ws-server"]
