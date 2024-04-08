FROM golang:1.18
WORKDIR /simple-ad-placement-service
COPY . .
RUN go mod download
RUN go build -o main .
EXPOSE 8080
CMD ["./main"]
