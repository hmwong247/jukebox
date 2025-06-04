### stage: build
FROM golang:1.24-alpine AS builder

WORKDIR /src

RUN apk update
RUN apk add npm

# frontend app
WORKDIR /src/app
COPY ./app/package.json .
RUN npm install

# web server
COPY go.mod go.sum .
RUN go mod download

# build all
WORKDIR /src
COPY . .
WORKDIR /src/app
RUN npm run build
WORKDIR /src
RUN CGO_ENABLED=0 go build -v -o ./main ./cmd/main.go

### stage: run
FROM scratch
COPY --from=builder /src/app/dist ./app/dist
COPY --from=builder /src/main ./main

EXPOSE 8080
CMD ["./main"]
