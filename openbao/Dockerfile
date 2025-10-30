# Multi-stage build for OpenBao namespace/key management tool
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/openbao-tool ./

FROM gcr.io/distroless/static:nonroot
USER nonroot:nonroot
WORKDIR /
COPY --from=build /bin/openbao-tool /openbao-tool
EXPOSE 8080
ENTRYPOINT ["/openbao-tool", "--server", "--listen", ":8080"]
