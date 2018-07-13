.PHONY: all clean install test

all:
	vgo build -o bin/sendmail

clean:
	rm bin/* -rf

install:
	./gi.sh

test:
	vgo test ./...
