export GOPATH=$(shell pwd)

all: search.go pkg
	go build search.go

pkg:
	go get golang.org/x/crypto/acme/autocert 

.PHONY: clean

clean: 
	rm search
