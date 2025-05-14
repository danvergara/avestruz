FROM golang:1.24-bookworm AS build

ARG TARGETOS
ARG TARGETARCH

RUN apt-get update -y \
  && apt-get clean

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o avestruz .

FROM gcr.io/distroless/base-debian11 AS build-release-stage

COPY --from=build /app/avestruz /bin/avestruz
