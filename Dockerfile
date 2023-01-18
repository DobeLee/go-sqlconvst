FROM golang

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

ENV GOPROXY https://proxy.golang.com.cn,direct

RUN go mod download

COPY *.go ./
COPY default.yaml ./
COPY my.sql ./

RUN go build -o ./go-sqlconvst

CMD ./go-sqlconvst