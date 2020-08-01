# Use the official Golang image to create a build artifact.
FROM golang:1.14 as builder

# Create and change to the build directory.
WORKDIR /build

# Download core dependencies.
COPY ./go.* ./
RUN go mod download

# Download template dependencies.
COPY ./template/src/go.* ./template/src/
RUN cd template/src; go mod download

# Copy core source files
COPY ./*.go ./

# Copy template source files
COPY ./template/src/ ./template/src/

# Build the binary.
RUN cd template/src; CGO_ENABLED=0 GOOS=linux go build -mod=readonly -o /build/bin/server

# Use the official Alpine image for a lean production container.
FROM alpine
RUN apk add --no-cache ca-certificates

# Create and change to the app directory.
WORKDIR /app

# Copy the binary from the builder stage.
COPY --from=builder /build/bin/server ./server

# Copy the graphql schema.
COPY ./template/schema.graphql ./schema.graphql

# Copy databse migrations.
COPY ./template/database/migrations ./database/migrations

# Run the web service on container startup.
ENTRYPOINT ["./server"]

# Default arguments
CMD ["--config", "env://CORE_CONFIG"]
