FROM golang:1 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-s -w" -o ipcoin -trimpath cmd/server/*.go

FROM alpine
COPY --from=builder /app/ipcoin /ipcoin
CMD ["/ipcoin"]
