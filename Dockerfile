FROM golang:1.27.1-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/url-short ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/url-short /url-short
EXPOSE 8080
ENTRYPOINT ["/url-short"]
