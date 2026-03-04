BINARY=ctfextension
CMD=./cmd/main.go

.PHONY: build run clean install release-linux release-mac release-windows all

build:
	go build -o $(BINARY) $(CMD)

run: build
	./$(BINARY)

# Install to /usr/local/bin so you can run `ctfextension` from anywhere
install: build
	sudo cp $(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed! You can now run: ctfextension <src> <dst>"

clean:
	rm -f $(BINARY) $(BINARY)-*

release-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY)-linux-amd64 $(CMD)
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BINARY)-linux-arm64 $(CMD)

release-mac:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY)-macos-amd64 $(CMD)
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BINARY)-macos-arm64 $(CMD)

release-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY)-windows-amd64.exe $(CMD)

all: release-linux release-mac release-windows
