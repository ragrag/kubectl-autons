.PHONY: build build-cross

build:
	go build -o bin/kubectl-autons main.go
	
build-cross:
	./build-cross