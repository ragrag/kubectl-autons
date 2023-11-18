.PHONY: build cross-compile

build:
	go build -ldflags="-w -s" -o bin/kubectl-autons src/main.go

cross-compile:
	./cross-compile