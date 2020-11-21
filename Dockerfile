FROM golang:1.15 as builder


WORKDIR /go/src/github.com/zerocube/roo

COPY go.mod go.sum ./

RUN go mod download -x

COPY . ./

ENV CGO_ENABLED=0

RUN go test -v ./... && go build -o /roo -v .

ENTRYPOINT ["go", "run", "."]

FROM scratch

# Copy across the CA certificate bundle
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.certs

# Copy across the compiled binary
COPY --from=builder /roo ./

ENTRYPOINT ["./roo"]
