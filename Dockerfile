FROM golang:1.24 as builder
WORKDIR /src
COPY . .
RUN go build -o /bin/server ./cmd/server

FROM gcr.io/distroless/base-debian12
COPY --from=builder /bin/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
