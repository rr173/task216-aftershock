FROM docker.m.daocloud.io/library/golang:1.26.3-bookworm

ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn
ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /app/aftershock ./cmd/aftershock

ENTRYPOINT ["/app/aftershock"]
CMD ["--smoke-test"]
