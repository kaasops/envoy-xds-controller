# Builds the e2e test dynamic module: docker buildx build --output type=local,dest=<dir> .
# golang:1.26 is the image the controller is built with, so CI has it already.
# Its glibc is 2.41, but the module needs symbols up to GLIBC_2.34 only, so it
# loads in Envoy images based on Ubuntu 22.04 (glibc 2.35).
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=1 go build -buildmode=c-shared -o /out/libe2e.so .

FROM scratch
COPY --from=build /out/libe2e.so /
