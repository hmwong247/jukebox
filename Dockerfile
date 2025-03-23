# build stage
# FROM golang:1.24 AS builder
#
# RUN apt update
# RUN apt install -y npm jq
#
# WORKDIR /build
#
# COPY go.mod go.sum .
# RUN go mod download
#
# COPY package.json .
# RUN npm install
#
# COPY . .
# RUN CGO_ENABLED=0 go build -v -o ./main ./cmd/main.go

# run stage
FROM golang:1.24-alpine

RUN apk update
RUN apk add jq npm

WORKDIR /app

COPY package.json .
RUN npm install

COPY go.mod go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -v -o ./main ./cmd/main.go

# third_party binary
# COPY --from=builder /build/third_party ./third_party/
# COPY --from=builder /build/node_modules ./node_modules/

# files
# COPY scripts ./scripts/
# COPY statics ./statics/
# COPY styles ./styles/
# COPY templates ./templates/

# binary
# COPY --from=builder /build/main ./main

EXPOSE 8080

CMD ["./main"]
