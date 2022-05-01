# rate-limiter-with-go

Reference: https://mauricio.github.io/2021/12/30/rate-limiting-in-go.html

## Environment
* Go v1.17.9
* Redis v7.0.0 (default address = "localhost:6379")

## Run this example
By default, this implementation is used a fixed-window strategy.
Clients are allowed to make 10 requests every minute. Once they 
go over 10 requests, application start denying the requests 
letting them know they’re over their quota and need to wait.

    go run main.go

    # execute this instruction many times
    curl --header "X-Forwarded-For: 127.0.0.1" localhost:8080/api/v1/hello