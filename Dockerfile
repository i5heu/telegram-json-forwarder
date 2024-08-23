FROM docker.io/golang:latest

WORKDIR /usr/src/app

COPY go.mod ./

RUN go mod tidy && go mod verify

COPY . .
RUN go build -v -o server ./main.go

EXPOSE 80

CMD ["/usr/src/app/server"]