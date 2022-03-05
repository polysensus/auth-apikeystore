# Go binaries are standalone, so use a multi-stage build to produce smaller images.
# Use base golang image from Docker Hub
FROM golang:1.17 as build

WORKDIR /build/apibin

# Install dependencies in go.mod and go.sum
COPY apibin/go.mod apibin/go.sum ./
RUN go mod download


WORKDIR /build/apihttp
COPY apihttp/go.mod apihttp/go.sum ./
RUN go mod download

WORKDIR /build/service
COPY service/go.mod service/go.sum ./
RUN go mod download


WORKDIR /build
# Copy rest of the application source code
COPY apibin/ ./apibin/
COPY apihttp/ ./apihttp/
COPY service/ ./service/

RUN find .

WORKDIR /build/service
# Skaffold passes in debug-oriented compiler flags
ARG SKAFFOLD_GO_GCFLAGS
RUN echo "Go gcflags: ${SKAFFOLD_GO_GCFLAGS}"
RUN go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -mod=readonly -v -o /build/server cmd/apikeystore/main.go

# Now create separate deployment image
FROM gcr.io/distroless/base

# Definition of this variable is used by 'skaffold debug' to identify a golang binary.
# Default behavior - a failure prints a stack trace for the current goroutine.
# See https://golang.org/pkg/runtime/
ENV GOTRACEBACK=single

WORKDIR /service
COPY --from=build /build/server ./
ENTRYPOINT ["/service/server"]
