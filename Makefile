.PHONY: all clean test

all:
	vgo build -o bin/sendmail

clean:
	rm bin/* -rf

test:
	vgo test ./...
