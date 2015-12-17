

all: client server

libs: lib/*

client: libs
	go build -o bin/client src/common.go src/client.go

server: libs
	go build -o bin/server src/common.go src/server.go

bin-dir:
	mkdir -p bin

clean:
	rm -rf bin
