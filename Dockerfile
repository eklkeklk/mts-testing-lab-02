FROM golang:1.22-bookworm

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY api_test.go ./

CMD ["go", "test", "-v", "-count=1", "-run", "^TestGraphQLHelloQuery$", "./..."]
