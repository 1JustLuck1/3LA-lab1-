FROM golang:1.23
WORKDIR /app
COPY . .
# RUN go mod download
# RUN go build -o server .
# EXPOSE 8080
# CMD ["./server"]

# Устанавливаем Air для горячей перезагрузки
RUN go install github.com/air-verse/air@latest

# Устанавливаем зависимости
RUN go mod download
RUN go build -o server .
EXPOSE 8080
# Запускаем Air
CMD ["air"]