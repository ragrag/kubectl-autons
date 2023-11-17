.PHONY: build build-cross

build:
	go build -ldflags="-w -s" -o bin/kubectl-autons main.go

build-cross:
	./build-cross