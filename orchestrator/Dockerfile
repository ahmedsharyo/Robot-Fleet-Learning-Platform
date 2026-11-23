FROM golang:1.23-alpine AS build
RUN apk add --no-cache build-base
WORKDIR /src
COPY orchestrator/go.mod orchestrator/go.sum* ./
RUN go mod download
COPY orchestrator ./
RUN CGO_ENABLED=1 go build -o /out/fleet ./cmd/server
FROM alpine:3.20
RUN apk add --no-cache ca-certificates sqlite-libs
COPY --from=build /out/fleet /usr/local/bin/fleet
EXPOSE 8080
ENTRYPOINT ["fleet"]
