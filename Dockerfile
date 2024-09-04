FROM golang:1.23 AS base

# Создаем образ для сборки
FROM base AS built

# Устанавливаем рабочую директорию
WORKDIR /go/app/api

# Копируем модульные файлы и загружаем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем остальной исходный код
COPY . .

# Устанавливаем переменную окружения для отключения cgo
ENV CGO_ENABLED=0
ENV API_SERVER_ADDR=:3000
ENV DATABASE_URL=postgres://user:password@db:5432/mydatabase

# Сборка приложения
RUN go get -d -v ./...
RUN go build -o /tmp/api-server ./*.go

# Используем минимальный образ для запуска
FROM busybox

# Копируем собранное приложение из образа сборки
COPY --from=built /tmp/api-server /usr/bin/api-server

# Команда по умолчанию для запуска приложения
CMD ["api-server", "start"]