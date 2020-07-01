# Use the official Golang image to create a build artifact.
FROM golang:1.14 as builder

# Create and change to the app directory.
WORKDIR /app

# Download application dependencies.
COPY ./src/go.* ./
RUN go mod download

# Copy local code to the container image.
COPY ./src ./

# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -o server

# Use the official Alpine image for a lean production container.
FROM alpine
RUN apk add --no-cache ca-certificates

# Create and change to the app directory.
WORKDIR /app

# Copy the binary from the builder stage.
COPY --from=builder /app/server ./server

# Copy the graphql schema.
COPY ./schema.graphql ./schema.graphql

# Run the web service on container startup.
ENTRYPOINT ["./server"]