FROM golang:1.19.4-alpine3.17
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN go build -o peer .
CMD ["/app/peer", "-secio",  "-l", "10000"]