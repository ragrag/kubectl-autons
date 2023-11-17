.PHONY: build run

build:
	go build -o bin/kubectl-autons main.go

build-cross:
	./build-cross