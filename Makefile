.PHONY: all clean

all:
	vgo build -o bin/sendmail

clean:
	rm bin/* -rf
