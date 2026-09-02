FROM golang:1.26.6-alpine AS build
WORKDIR /src
ARG TARGET=api
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /visionops ./cmd/${TARGET}

FROM alpine:3.20
COPY --from=build /visionops /visionops
COPY web /web
COPY migrations /migrations
COPY fixtures /fixtures
EXPOSE 8080
ENTRYPOINT ["/visionops"]
