FROM golang:1.22 as builder
COPY go.mod go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0  go build -o /bin/dispatch-echo ./cmd/dispatch-echo/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/dispatch-echo /bin/dispatch-echo
EXPOSE 50051
CMD ["/bin/dispatch-echo"]