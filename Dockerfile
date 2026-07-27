FROM golang:1.25

WORKDIR /app

COPY . .

RUN go build -o company-service ./cmd/server

EXPOSE 8080

CMD ["./company-service"]