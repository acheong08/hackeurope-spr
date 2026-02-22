FROM golang:1.22-bullseye

# Install node + npm from Debian repo (stable & reliable)
RUN apt-get update && \
    apt-get install -y nodejs npm && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 8080

CMD ["go", "run", "./cmd/server/"]
