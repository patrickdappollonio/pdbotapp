FROM golang AS builder
WORKDIR /go/src/github.com/patrickdappollonio/pdbotapp
ADD . .
RUN go build -a -tags netgo -ldflags '-s -w' -o build/pdbotapp
RUN mkdir -p build/templates/ && cp -R templates build/

FROM alpine
COPY --from=builder /go/src/github.com/patrickdappollonio/pdbotapp/build/ /app
CMD ./app/pdbotapp
