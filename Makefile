.PHONY: all clean install test

all:
	go build -o bin/sendmail

clean:
	rm bin/* -rf

install:
	./gi.sh

test:
	go test ./...
