FROM golang:1.23.2

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./

RUN cd ./cmd/web && go build -o /transcribe-to-notion

EXPOSE 4000

CMD ["/transcribe-to-notion"]